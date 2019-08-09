package remotes

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/containerd/remotes"
	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/distribution/reference"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// FixupBundle checks that all the references are present in the referenced repository, otherwise it will mount all
// the manifests to that repository. The bundle is then patched with the new digested references.
func FixupBundle(ctx context.Context, b *bundle.Bundle, ref reference.Named, resolver remotes.Resolver, opts ...FixupOption) error {
	logger := log.G(ctx)
	logger.Debugf("Fixing up bundle %s", ref)
	cfg, err := newFixupConfig(b, ref, resolver, opts...)
	if err != nil {
		return err
	}
	events := make(chan FixupEvent)
	eventLoopDone := make(chan struct{})
	defer func() {
		close(events)
		// wait for all queued events to be treated
		<-eventLoopDone
	}()
	go func() {
		defer close(eventLoopDone)
		for ev := range events {
			cfg.eventCallback(ev)
		}
	}()

	if len(b.InvocationImages) != 1 {
		return fmt.Errorf("only one invocation image supported for bundle %q", ref)
	}
	if b.InvocationImages[0].BaseImage, err = fixupImage(ctx, b.InvocationImages[0].BaseImage, cfg, events, cfg.invocationImagePlatformFilter); err != nil {
		return err
	}
	for name, original := range b.Images {
		if original.BaseImage, err = fixupImage(ctx, original.BaseImage, cfg, events, cfg.componentImagePlatformFilter); err != nil {
			return err
		}
		b.Images[name] = original
	}
	logger.Debug("Bundle fixed")
	return nil
}

func fixupImage(ctx context.Context, baseImage bundle.BaseImage, cfg fixupConfig, events chan<- FixupEvent, platformFilter platforms.Matcher) (_ bundle.BaseImage, retErr error) {
	log.G(ctx).Debugf("Fixing image %s", baseImage.Image)
	ctx = withMutedContext(ctx)
	progress := &progress{}
	originalSource := baseImage.Image
	notifyEvent := func(eventType FixupEventType, message string, err error) {
		events <- FixupEvent{
			DestinationRef: cfg.targetRef,
			SourceImage:    originalSource,
			EventType:      eventType,
			Message:        message,
			Error:          err,
			Progress:       progress.snapshot(),
		}
	}
	defer func() {
		if retErr != nil {
			notifyEvent(FixupEventTypeCopyImageEnd, "", retErr)
		}
	}()
	notifyEvent(FixupEventTypeCopyImageStart, "", nil)
	fixupInfo, err := fixupBaseImage(ctx, &baseImage, cfg.targetRef, cfg.resolver)
	if err != nil {
		return bundle.BaseImage{}, err
	}

	if fixupInfo.sourceRef.Name() == fixupInfo.targetRepo.Name() {
		notifyEvent(FixupEventTypeCopyImageEnd, "Nothing to do: image reference is already present in repository"+fixupInfo.targetRepo.String(), nil)
		return baseImage, nil
	}
	sourceRepoOnly, err := reference.ParseNormalizedNamed(fixupInfo.sourceRef.Name())
	if err != nil {
		return bundle.BaseImage{}, err
	}
	f, err := cfg.resolver.Fetcher(ctx, sourceRepoOnly.Name())
	if err != nil {
		return bundle.BaseImage{}, err
	}
	sourceFetcher := newSourceFetcherWithLocalData(f)
	if err := fixupPlatforms(ctx, &baseImage, &fixupInfo, sourceFetcher, platformFilter); err != nil {
		return bundle.BaseImage{}, err
	}

	// Prepare the copier
	copier, err := newDescriptorCopier(ctx, cfg.resolver, sourceFetcher, fixupInfo.targetRepo.String(), notifyEvent, fixupInfo.sourceRef)
	if err != nil {
		return bundle.BaseImage{}, err
	}
	descriptorContentHandler := &descriptorContentHandler{
		descriptorCopier: copier,
		targetRepo:       fixupInfo.targetRepo.String(),
	}
	ctx, cancel := context.WithCancel(ctx)
	scheduler := newErrgroupScheduler(ctx, cfg.maxConcurrentJobs, cfg.jobsBufferLength)
	defer func() {
		cancel()
		scheduler.drain()
	}()
	walker := newManifestWalker(notifyEvent, scheduler, progress, descriptorContentHandler)
	walkerDep := walker.walk(scheduler.ctx(), fixupInfo.resolvedDescriptor, nil)
	if err = walkerDep.wait(); err != nil {
		return bundle.BaseImage{}, err
	}
	notifyEvent(FixupEventTypeCopyImageEnd, "", nil)
	return baseImage, nil
}

func fixupPlatforms(ctx context.Context, baseImage *bundle.BaseImage, fixupInfo *imageFixupInfo, sourceFetcher sourceFetcherAdder, filter platforms.Matcher) error {
	if filter == nil ||
		(fixupInfo.resolvedDescriptor.MediaType != ocischemav1.MediaTypeImageIndex && fixupInfo.resolvedDescriptor.MediaType != images.MediaTypeDockerSchema2ManifestList) {
		// no platform filter if platform is empty, or if the descriptor is not an OCI Index / Docker Manifest list
		return nil
	}

	reader, err := sourceFetcher.Fetch(ctx, fixupInfo.resolvedDescriptor)
	if err != nil {
		return err
	}
	defer reader.Close()

	manifestBytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	var manifestList typelessManifestList
	if err := json.Unmarshal(manifestBytes, &manifestList); err != nil {
		return err
	}
	var validManifests []typelessDescriptor
	for _, d := range manifestList.Manifests {
		if d.Platform != nil && filter.Match(*d.Platform) {
			validManifests = append(validManifests, d)
		}
	}
	if len(validManifests) == 0 {
		return fmt.Errorf("no descriptor matching the platform filter found in %q", fixupInfo.sourceRef)
	}
	manifestList.Manifests = validManifests
	manifestBytes, err = json.Marshal(&manifestList)
	if err != nil {
		return err
	}
	d := sourceFetcher.Add(manifestBytes)
	descriptor := fixupInfo.resolvedDescriptor
	descriptor.Digest = d
	descriptor.Size = int64(len(manifestBytes))
	fixupInfo.resolvedDescriptor = descriptor
	newRef, err := reference.WithDigest(fixupInfo.targetRepo, d)
	if err != nil {
		return err
	}
	baseImage.Image = newRef.String()
	return nil
}

func fixupBaseImage(ctx context.Context,
	baseImage *bundle.BaseImage,
	targetRef reference.Named, //nolint: interfacer
	resolver remotes.Resolver) (imageFixupInfo, error) {
	err := checkBaseImage(baseImage)
	if err != nil {
		err := fmt.Errorf("invalid image %q: %s", baseImage.Image, err)
		return imageFixupInfo{}, err
	}
	targetRepoOnly, err := reference.ParseNormalizedNamed(targetRef.Name())
	if err != nil {
		return imageFixupInfo{}, err
	}
	sourceImageRef, err := reference.ParseNormalizedNamed(baseImage.Image)
	if err != nil {
		err = fmt.Errorf("%q is not a valid image reference for %q", baseImage.Image, targetRef)
		return imageFixupInfo{}, err
	}
	sourceImageRef = reference.TagNameOnly(sourceImageRef)
	_, descriptor, err := resolver.Resolve(ctx, sourceImageRef.String())
	if err != nil {
		err = fmt.Errorf("failed to resolve %q, push the image to the registry before pushing the bundle: %s", sourceImageRef, err)
		return imageFixupInfo{}, err
	}
	digested, err := reference.WithDigest(targetRepoOnly, descriptor.Digest)
	if err != nil {
		return imageFixupInfo{}, err
	}
	baseImage.Image = reference.FamiliarString(digested)
	baseImage.MediaType = descriptor.MediaType
	baseImage.Size = uint64(descriptor.Size)
	return imageFixupInfo{
		resolvedDescriptor: descriptor,
		sourceRef:          sourceImageRef,
		targetRepo:         targetRepoOnly,
	}, nil
}

func checkBaseImage(baseImage *bundle.BaseImage) error {
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
