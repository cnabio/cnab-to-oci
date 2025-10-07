package remotes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/distribution/reference"
	"github.com/opencontainers/go-digest"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/sync/errgroup"
)

const (
	// labelDistributionSource describes the source blob comes from.
	// This label comes from containerd: https://github.com/containerd/containerd/blob/master/remotes/docker/handler.go#L35
	labelDistributionSource = "containerd.io/distribution.source"
)

func newDescriptorCopier(ctx context.Context, resolver remotes.Resolver,
	sourceFetcher remotes.Fetcher, targetRepo string,
	eventNotifier eventNotifier, originalSource reference.Named) (*descriptorCopier, error) {
	destPusher, err := resolver.Pusher(ctx, targetRepo)
	if err != nil {
		return nil, err
	}
	return &descriptorCopier{
		sourceFetcher:  sourceFetcher,
		targetPusher:   destPusher,
		eventNotifier:  eventNotifier,
		resolver:       resolver,
		originalSource: originalSource,
	}, nil
}

type descriptorCopier struct {
	sourceFetcher  remotes.Fetcher
	targetPusher   remotes.Pusher
	eventNotifier  eventNotifier
	resolver       remotes.Resolver
	originalSource reference.Named
}

func (h *descriptorCopier) Handle(ctx context.Context, desc *descriptorProgress) (retErr error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if len(desc.URLs) > 0 {
		desc.markDone()
		desc.setAction("Skip (foreign layer)")
		return nil
	}
	desc.setAction("Copy")
	h.eventNotifier.reportProgress(nil)
	defer func() {
		if retErr != nil {
			desc.setError(retErr)
		}
		h.eventNotifier.reportProgress(retErr)
	}()
	writer, err := pushWithAnnotation(ctx, h.targetPusher, h.originalSource, desc.Descriptor)
	if errors.Is(err, errdefs.ErrAlreadyExists) {
		desc.markDone()
		if strings.Contains(err.Error(), "mounted") {
			desc.setAction("Mounted")
		}
		return nil
	}
	if err != nil {
		return err
	}
	defer writer.Close()
	reader, err := h.sourceFetcher.Fetch(ctx, desc.Descriptor)
	if err != nil {
		return err
	}
	defer reader.Close()
	err = content.Copy(ctx, writer, reader, desc.Size, desc.Digest)
	if errors.Is(err, errdefs.ErrAlreadyExists) {
		err = nil
	}
	if err == nil {
		desc.markDone()
	}
	return err
}

func pushWithAnnotation(ctx context.Context, pusher remotes.Pusher, ref reference.Named, desc ocischemav1.Descriptor) (content.Writer, error) {
	// Add the distribution source annotation to help containerd
	// mount instead of push when possible.
	repo := fmt.Sprintf("%s.%s", labelDistributionSource, reference.Domain(ref))
	desc.Annotations = map[string]string{
		repo: reference.FamiliarName(ref),
	}
	return pusher.Push(ctx, desc)
}

func isManifest(mediaType string) bool {
	return mediaType == images.MediaTypeDockerSchema1Manifest ||
		mediaType == images.MediaTypeDockerSchema2Manifest ||
		mediaType == images.MediaTypeDockerSchema2ManifestList ||
		mediaType == ocischemav1.MediaTypeImageIndex ||
		mediaType == ocischemav1.MediaTypeImageManifest
}

type imageContentProvider struct {
	fetcher remotes.Fetcher
}

func (p *imageContentProvider) ReaderAt(ctx context.Context, desc ocischemav1.Descriptor) (content.ReaderAt, error) {
	rc, err := p.fetcher.Fetch(ctx, desc)
	if err != nil {
		return nil, err
	}
	return &remoteReaderAt{ReadCloser: rc, currentOffset: 0, size: desc.Size}, nil
}

type remoteReaderAt struct {
	io.ReadCloser
	currentOffset int64
	size          int64
}

func (r *remoteReaderAt) Size() int64 {
	return r.size
}

func (r *remoteReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off != r.currentOffset {
		return 0, fmt.Errorf("at the moment this reader only supports offset at %d, requested offset was %d", r.currentOffset, off)
	}
	n, err := r.Read(p)
	r.currentOffset += int64(n)
	if err == io.EOF && n == len(p) {
		return n, nil
	}
	if err != nil || n == len(p) {
		return n, err
	}
	n2, err := r.ReadAt(p[n:], r.currentOffset)
	n += n2
	return n, err
}

type descriptorContentHandler struct {
	descriptorCopier *descriptorCopier
	targetRepo       string

	// Keep track of which layers we have copied for this image
	// so that we can avoid copying the same layer more than once.
	layersScheduled map[digest.Digest]struct{}
}

func (h *descriptorContentHandler) createCopyTask(ctx context.Context, descProgress *descriptorProgress) (func(ctx context.Context) error, error) {
	if _, scheduled := h.layersScheduled[descProgress.Digest]; scheduled {
		return func(_ context.Context) error {
			// Skip. We have already scheduled a copy of this layer
			return nil
		}, nil
	}

	// Mark that we have scheduled this layer. Some images can have a layer duplicated
	// within the image and attempts to copy the same layer multiple times results in
	// unexpected size errors when the later copy tasks try to copy an existing layer.
	if h.layersScheduled == nil {
		h.layersScheduled = make(map[digest.Digest]struct{}, 1)
	}
	h.layersScheduled[descProgress.Digest] = struct{}{}

	copyOrMountWorkItem := func(ctx context.Context) error {
		return h.descriptorCopier.Handle(ctx, descProgress)
	}
	if !isManifest(descProgress.MediaType) {
		return copyOrMountWorkItem, nil
	}
	_, _, err := h.descriptorCopier.resolver.Resolve(ctx, fmt.Sprintf("%s@%s", h.targetRepo, descProgress.Digest))
	if err == nil {
		descProgress.setAction("Skip (already present)")
		descProgress.markDone()
		return nil, errdefs.ErrAlreadyExists
	}
	return copyOrMountWorkItem, nil
}

type manifestWalker struct {
	getChildren    images.HandlerFunc
	eventNotifier  eventNotifier
	scheduler      scheduler
	progress       *progress
	contentHandler *descriptorContentHandler
}

func newManifestWalker(
	eventNotifier eventNotifier,
	scheduler scheduler,
	progress *progress,
	descriptorContentHandler *descriptorContentHandler) *manifestWalker {
	sourceFetcher := descriptorContentHandler.descriptorCopier.sourceFetcher
	return &manifestWalker{
		eventNotifier:  eventNotifier,
		getChildren:    images.ChildrenHandler(&imageContentProvider{sourceFetcher}),
		scheduler:      scheduler,
		progress:       progress,
		contentHandler: descriptorContentHandler,
	}
}

type copyTask struct {
	digest   digest.Digest
	copyTask func(ctx context.Context) error
	depth    int
}

func (w *manifestWalker) collectCopyTasks(ctx context.Context, desc ocischemav1.Descriptor, parent *descriptorProgress, depth int) ([]copyTask, error) {
	descProgress := &descriptorProgress{
		Descriptor: desc,
	}
	if parent != nil {
		parent.addChild(descProgress)
	} else {
		w.progress.addRoot(descProgress)
	}

	var allItems []copyTask
	copyOrMountWorkItem, err := w.contentHandler.createCopyTask(ctx, descProgress)
	if errors.Is(err, errdefs.ErrAlreadyExists) {
		w.eventNotifier.reportProgress(nil)
		return nil, nil
	}
	if err != nil {
		w.eventNotifier.reportProgress(err)
		return nil, err
	}
	allItems = append(allItems, copyTask{
		desc.Digest,
		copyOrMountWorkItem,
		depth,
	})
	children, err := w.getChildren.Handle(ctx, desc)
	if err != nil {
		return nil, err
	}

	for _, c := range children {
		childCopyTasks, err := w.collectCopyTasks(ctx, c, descProgress, depth+1)
		if err != nil {
			return nil, err
		}
		allItems = append(allItems, childCopyTasks...)
	}

	return allItems, nil
}

func (w *manifestWalker) walk(ctx context.Context, desc ocischemav1.Descriptor) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	tasks, err := w.collectCopyTasks(ctx, desc, nil, 0)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		return nil
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].depth > tasks[j].depth
	})

	workGroup, c := errgroup.WithContext(ctx)
	workGroup.SetLimit(4)
	lastDepth := tasks[0].depth
	for _, task := range tasks {
		if task.depth != lastDepth {
			err = workGroup.Wait()
			if err != nil {
				return err
			}
			workGroup, c = errgroup.WithContext(ctx)
			workGroup.SetLimit(4)
		}
		workGroup.Go(func() error {
			select {
			case <-c.Done():
				return c.Err()
			default:
			}
			err = task.copyTask(c)
			if err != nil {
				return err
			}
			return nil
		})
		lastDepth = task.depth
	}

	return workGroup.Wait()
}

type eventNotifier func(eventType FixupEventType, message string, err error)

func (n eventNotifier) reportProgress(err error) {
	n(FixupEventTypeProgress, "", err)
}
