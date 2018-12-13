package remotes

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/distribution/reference"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

// FixupBundle TODO
func FixupBundle(ctx context.Context, b *bundle.Bundle, ref reference.Named, resolver docker.ResolverBlobMounter) error {
	if len(b.InvocationImages) != 1 {
		return fmt.Errorf("only one invocation image supported for bundle %q", ref)
	}
	var err error
	if b.InvocationImages[0].BaseImage, err = fixupImage(ctx, b.InvocationImages[0].BaseImage, ref, resolver); err != nil {
		return err
	}
	for name, original := range b.Images {
		if original.BaseImage, err = fixupImage(ctx, original.BaseImage, ref, resolver); err != nil {
			return err
		}
		b.Images[name] = original
	}
	return nil
}

type imageCopier struct {
	sourceFetcher remotes.Fetcher
	targetPusher  remotes.Pusher
}

func (h *imageCopier) Handle(ctx context.Context, desc ocischemav1.Descriptor) (err error) {
	fmt.Fprintf(os.Stderr, "Copying descriptor %s with media type %s (size: %d)\n", desc.Digest, desc.MediaType, desc.Size)
	reader, err := h.sourceFetcher.Fetch(ctx, desc)
	if err != nil {
		return err
	}
	defer reader.Close()
	writer, err := h.targetPusher.Push(ctx, desc)
	if err != nil {
		if errors.Cause(err) == errdefs.ErrAlreadyExists {
			return nil
		}
		return err
	}
	defer writer.Close()
	err = content.Copy(ctx, writer, reader, desc.Size, desc.Digest)
	if errors.Cause(err) == errdefs.ErrAlreadyExists {
		return nil
	}
	return err
}

type imageMounter struct {
	imageCopier
	sourceRepo    string
	targetMounter docker.BlobMounter
}

func isManifest(mediaType string) bool {
	return mediaType == images.MediaTypeDockerSchema1Manifest ||
		mediaType == images.MediaTypeDockerSchema2Manifest ||
		mediaType == images.MediaTypeDockerSchema2ManifestList ||
		mediaType == ocischemav1.MediaTypeImageIndex ||
		mediaType == ocischemav1.MediaTypeImageManifest
}

func (h *imageMounter) Handle(ctx context.Context, desc ocischemav1.Descriptor) error {
	if isManifest(desc.MediaType) {
		// manifests are copied
		return h.imageCopier.Handle(ctx, desc)
	}
	fmt.Fprintf(os.Stderr, "Mounting descriptor %s with media type %s (size: %d)\n", desc.Digest, desc.MediaType, desc.Size)
	err := h.targetMounter.MountBlob(ctx, desc, h.sourceRepo)
	if errors.Cause(err) == errdefs.ErrAlreadyExists {
		return nil
	}
	return err
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
	return n, err
}

type descriptorAccumulator struct {
	descriptors []ocischemav1.Descriptor
}

func (a *descriptorAccumulator) Handle(ctx context.Context, desc ocischemav1.Descriptor) ([]ocischemav1.Descriptor, error) {
	descs := make([]ocischemav1.Descriptor, len(a.descriptors)+1)
	descs[0] = desc
	for ix, d := range a.descriptors {
		descs[ix+1] = d
	}
	a.descriptors = descs
	return nil, nil
}

func fixupImage(ctx context.Context, baseImage bundle.BaseImage, ref reference.Named, resolver docker.ResolverBlobMounter) (bundle.BaseImage, error) {
	fmt.Fprintf(os.Stderr, "Ensuring image %s is present in repo %s", baseImage.Image, ref.Name())
	if err := checkBaseImage(baseImage); err != nil {
		return bundle.BaseImage{}, fmt.Errorf("invalid image %q: %s", ref, err)
	}
	repoOnly, err := reference.ParseNormalizedNamed(ref.Name())
	if err != nil {
		return bundle.BaseImage{}, err
	}
	imageRef, err := reference.ParseNormalizedNamed(baseImage.Image)
	if err != nil {
		return bundle.BaseImage{}, fmt.Errorf("%q is not a valid image reference for %q", baseImage.Image, ref)
	}
	_, descriptor, err := resolver.Resolve(ctx, imageRef.String())
	if err != nil {
		return bundle.BaseImage{}, err
	}
	digested, err := reference.WithDigest(repoOnly, descriptor.Digest)
	if err != nil {
		return bundle.BaseImage{}, err
	}
	baseImage.Image = reference.FamiliarString(digested)
	baseImage.MediaType = descriptor.MediaType
	baseImage.Size = uint64(descriptor.Size)
	if imageRef.Name() == ref.Name() {
		return baseImage, nil
	}
	sourceRepoOnly, err := reference.ParseNormalizedNamed(imageRef.Name())
	if err != nil {
		return bundle.BaseImage{}, err
	}
	sourceFetcher, err := resolver.Fetcher(ctx, sourceRepoOnly.String())
	if err != nil {
		return bundle.BaseImage{}, err
	}
	destPusher, err := resolver.Pusher(ctx, repoOnly.String())
	if err != nil {
		return bundle.BaseImage{}, err
	}
	destMounter, err := resolver.BlobMounter(ctx, repoOnly.String())
	if err != nil {
		return bundle.BaseImage{}, err
	}
	accumulator := &descriptorAccumulator{}

	if err := images.Walk(ctx, images.Handlers(accumulator, images.ChildrenHandler(&imageContentProvider{sourceFetcher})), descriptor); err != nil {
		return bundle.BaseImage{}, err
	}
	if reference.Domain(imageRef) == reference.Domain(ref) {
		mounter := &imageMounter{
			imageCopier: imageCopier{
				sourceFetcher: sourceFetcher,
				targetPusher:  destPusher,
			},
			targetMounter: destMounter,
			sourceRepo:    sourceRepoOnly.Name(),
		}
		for _, d := range accumulator.descriptors {
			if err := mounter.Handle(ctx, d); err != nil {
				return bundle.BaseImage{}, err
			}
		}
	} else {
		copier := &imageCopier{
			sourceFetcher: sourceFetcher,
			targetPusher:  destPusher,
		}

		for _, d := range accumulator.descriptors {
			if err := copier.Handle(ctx, d); err != nil {
				return bundle.BaseImage{}, err
			}
		}
	}

	return baseImage, nil
}

func checkBaseImage(baseImage bundle.BaseImage) error {
	switch baseImage.ImageType {
	case "docker":
	case "oci":
	case "":
		baseImage.ImageType = "oci"
	default:
		return fmt.Errorf("image type %q is not supported", baseImage.ImageType)
	}

	switch baseImage.MediaType {
	case ocischemav1.MediaTypeImageIndex:
	case ocischemav1.MediaTypeImageManifest:
	case images.MediaTypeDockerSchema2Manifest:
	case images.MediaTypeDockerSchema2ManifestList:
	case "":
	default:
		return fmt.Errorf("image media type %q is not supported", baseImage.ImageType)
	}

	return nil
}
