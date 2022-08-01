package remotes

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-to-oci/relocation"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/containerd/remotes"
	"github.com/docker/distribution/reference"
	"github.com/hashicorp/go-multierror"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// FixupBundle checks that all the references are present in the referenced repository, otherwise it will mount all
// the manifests to that repository. The bundle is then patched with the new digested references.
func FixupBundle(ctx context.Context, b *bundle.Bundle, ref reference.Named, resolver remotes.Resolver, opts ...FixupOption) (relocation.ImageRelocationMap, error) {
	logger := log.G(ctx)
	logger.Debugf("Fixing up bundle %s", ref)

	// Configure the fixup and the event loop
	cfg, err := newFixupConfig(b, ref, resolver, opts...)
	if err != nil {
		return nil, err
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

	// Fixup invocation images
	if len(b.InvocationImages) != 1 {
		return nil, fmt.Errorf("only one invocation image supported for bundle %q", ref)
	}

	relocationMap := cfg.relocationMap
	if err := fixupImage(ctx, "InvocationImage", &b.InvocationImages[0].BaseImage, relocationMap, cfg, events, cfg.invocationImagePlatformFilter); err != nil {
		return nil, err
	}
	// Fixup images
	for name, original := range b.Images {
		if err := fixupImage(ctx, name, &original.BaseImage, relocationMap, cfg, events, cfg.componentImagePlatformFilter); err != nil {
			return nil, err
		}
		b.Images[name] = original
	}

	logger.Debug("Bundle fixed")
	return relocationMap, nil
}

func fixupImage(
	ctx context.Context,
	name string,
	baseImage *bundle.BaseImage,
	relocationMap relocation.ImageRelocationMap,
	cfg fixupConfig,
	events chan<- FixupEvent,
	platformFilter platforms.Matcher) error {

	// Fixup the base image, using the relocated base image if available
	sourceImage := *baseImage
	if relocatedBaseImage, ok := relocationMap[baseImage.Image]; ok {
		sourceImage.Image = relocatedBaseImage
	}

	log.G(ctx).Debugf("Updating entry in relocation map for %q", baseImage.Image)
	ctx = withMutedContext(ctx)
	notifyEvent, progress := makeEventNotifier(events, sourceImage.Image, cfg.targetRef)

	notifyEvent(FixupEventTypeCopyImageStart, "", nil)
	fixupInfo, pushed, err := fixupBaseImage(ctx, name, &sourceImage, cfg)
	if err != nil {
		return notifyError(notifyEvent, err)
	}
	// Update the relocation map with the original image name and the digested reference of the image pushed inside the bundle repository
	newRef, err := reference.WithDigest(fixupInfo.targetRepo, fixupInfo.resolvedDescriptor.Digest)
	if err != nil {
		return err
	}

	relocationMap[baseImage.Image] = newRef.String()

	// if the autoUpdateBundle flag is passed, mutate the bundle with the resolved digest, mediaType, and size
	if cfg.autoBundleUpdate {
		baseImage.Digest = fixupInfo.resolvedDescriptor.Digest.String()
		baseImage.Size = uint64(fixupInfo.resolvedDescriptor.Size)
		baseImage.MediaType = fixupInfo.resolvedDescriptor.MediaType
	} else {
		if baseImage.Digest != fixupInfo.resolvedDescriptor.Digest.String() {
			return fmt.Errorf("image %q digest differs %q after fixup: %q", baseImage.Image, baseImage.Digest, fixupInfo.resolvedDescriptor.Digest.String())
		}
		if baseImage.Size != uint64(fixupInfo.resolvedDescriptor.Size) {
			return fmt.Errorf("image %q size differs %d after fixup: %d", baseImage.Image, baseImage.Size, fixupInfo.resolvedDescriptor.Size)
		}
		if baseImage.MediaType != fixupInfo.resolvedDescriptor.MediaType {
			return fmt.Errorf("image %q media type differs %q after fixup: %q", baseImage.Image, baseImage.MediaType, fixupInfo.resolvedDescriptor.MediaType)
		}
	}

	if pushed {
		notifyEvent(FixupEventTypeCopyImageEnd, "Image has been pushed for service "+name, nil)
		return nil
	}

	if fixupInfo.sourceRef.Name() == fixupInfo.targetRepo.Name() {
		notifyEvent(FixupEventTypeCopyImageEnd, "Nothing to do: image reference is already present in repository"+fixupInfo.targetRepo.String(), nil)
		return nil
	}

	sourceFetcher, err := makeSourceFetcher(ctx, cfg.resolver, fixupInfo.sourceRef.Name())
	if err != nil {
		return notifyError(notifyEvent, err)
	}

	// Fixup platforms
	if err := fixupPlatforms(ctx, baseImage, relocationMap, &fixupInfo, sourceFetcher, platformFilter); err != nil {
		return notifyError(notifyEvent, err)
	}

	// Prepare and run the copier
	walkerDep, cleaner, err := makeManifestWalker(ctx, sourceFetcher, notifyEvent, cfg, fixupInfo, progress)
	if err != nil {
		return notifyError(notifyEvent, err)
	}
	defer cleaner()
	if err = walkerDep.wait(); err != nil {
		return notifyError(notifyEvent, err)
	}

	notifyEvent(FixupEventTypeCopyImageEnd, "", nil)
	return nil
}

func fixupPlatforms(ctx context.Context,
	baseImage *bundle.BaseImage,
	relocationMap relocation.ImageRelocationMap,
	fixupInfo *imageFixupInfo,
	sourceFetcher sourceFetcherAdder,
	filter platforms.Matcher) error {

	logger := log.G(ctx)
	logger.Debugf("Fixup platforms for image %v, with relocation map %v", baseImage, relocationMap)
	if filter == nil ||
		(fixupInfo.resolvedDescriptor.MediaType != ocischemav1.MediaTypeImageIndex &&
			fixupInfo.resolvedDescriptor.MediaType != images.MediaTypeDockerSchema2ManifestList) {
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

	return nil
}

func fixupBaseImage(ctx context.Context, name string, baseImage *bundle.BaseImage, cfg fixupConfig) (imageFixupInfo, bool, error) {
	// Check image references
	if err := checkBaseImage(baseImage); err != nil {
		return imageFixupInfo{}, false, fmt.Errorf("invalid image %q for service %q: %s", baseImage.Image, name, err)
	}
	targetRepoOnly, err := reference.ParseNormalizedNamed(cfg.targetRef.Name())
	if err != nil {
		return imageFixupInfo{}, false, err
	}

	fixups := []func(context.Context, reference.Named, *bundle.BaseImage, fixupConfig) (imageFixupInfo, bool, bool, error){
		pushByDigest,
		resolveImageInRelocationMap,
		resolveImage,
		pushLocalImage,
	}

	var bigErr *multierror.Error
	for _, f := range fixups {
		info, pushed, ok, err := f(ctx, targetRepoOnly, baseImage, cfg)
		if err != nil {
			log.G(ctx).Debug(err)
			// do not stop trying fixups after the first error. Only report the errors if all fixups were unable to push the image.
			bigErr = multierror.Append(bigErr, fmt.Errorf("failed to fixup the image %s for service %q: %v", baseImage.Image, name, err))
		}
		if ok {
			return info, pushed, nil
		}
	}

	return imageFixupInfo{}, false, bigErr.ErrorOrNil()
}

func pushByDigest(ctx context.Context, target reference.Named, baseImage *bundle.BaseImage, cfg fixupConfig) (imageFixupInfo, bool, bool, error) {
	if baseImage.Image != "" || !cfg.pushImages {
		return imageFixupInfo{}, false, false, nil
	}
	descriptor, err := pushImageToTarget(ctx, baseImage.Digest, cfg)
	if err != nil {
		return imageFixupInfo{}, false, false, fmt.Errorf("failed to push digested image %s@%s to target %s: %v", baseImage.Image, baseImage.Digest, target, err)
	}
	return imageFixupInfo{
		targetRepo:         target,
		sourceRef:          nil,
		resolvedDescriptor: descriptor,
	}, true, true, nil
}

func resolveImage(ctx context.Context, target reference.Named, baseImage *bundle.BaseImage, cfg fixupConfig) (imageFixupInfo, bool, bool, error) {
	sourceImageRef, err := ref(baseImage.Image)
	if err != nil {
		return imageFixupInfo{}, false, false, fmt.Errorf("failed to resolve image: invalid source ref %s: %w", baseImage.Image, err)
	}
	_, descriptor, err := cfg.resolver.Resolve(ctx, sourceImageRef.String())
	if err != nil {
		return imageFixupInfo{}, false, false, fmt.Errorf("failed to resolve image %s: %w", sourceImageRef.String(), err)
	}
	return imageFixupInfo{
		targetRepo:         target,
		sourceRef:          sourceImageRef,
		resolvedDescriptor: descriptor,
	}, false, true, nil
}

func resolveImageInRelocationMap(ctx context.Context, target reference.Named, baseImage *bundle.BaseImage, cfg fixupConfig) (imageFixupInfo, bool, bool, error) {
	sourceImageRef, err := ref(baseImage.Image)
	if err != nil {
		return imageFixupInfo{}, false, false, fmt.Errorf("failed to resolve image in relocation map: invalid source ref %s: %v", baseImage.Image, err)
	}
	relocatedRef, ok := cfg.relocationMap[baseImage.Image]
	if !ok {
		return imageFixupInfo{}, false, false, nil
	}
	relocatedImageRef, err := ref(relocatedRef)
	if err != nil {
		return imageFixupInfo{}, false, false, fmt.Errorf("failed to resolve image in relocation map: invalid target ref %s: %v", relocatedRef, err)
	}
	_, descriptor, err := cfg.resolver.Resolve(ctx, relocatedImageRef.String())
	if err != nil {
		return imageFixupInfo{}, false, false, err
	}
	return imageFixupInfo{
		targetRepo:         target,
		sourceRef:          sourceImageRef,
		resolvedDescriptor: descriptor,
	}, false, true, nil
}

func pushLocalImage(ctx context.Context, target reference.Named, baseImage *bundle.BaseImage, cfg fixupConfig) (imageFixupInfo, bool, bool, error) {
	if !cfg.pushImages {
		return imageFixupInfo{}, false, false, nil
	}
	sourceImageRef, err := ref(baseImage.Image)
	if err != nil {
		return imageFixupInfo{}, false, false, fmt.Errorf("failed to push local image: invalid source ref %s: %v", baseImage.Image, err)
	}
	descriptor, err := pushImageToTarget(ctx, baseImage.Image, cfg)
	if err != nil {
		return imageFixupInfo{}, false, false, fmt.Errorf("failed to push local image %s: %v", baseImage.Image, err)
	}
	return imageFixupInfo{
		targetRepo:         target,
		sourceRef:          sourceImageRef,
		resolvedDescriptor: descriptor,
	}, true, true, nil
}

func ref(str string) (reference.Named, error) {
	r, err := reference.ParseNormalizedNamed(str)
	if err != nil {
		return nil, fmt.Errorf("%q is not a valid reference: %v", str, err)
	}
	return reference.TagNameOnly(r), nil
}

// pushImageToTarget pushes the image from the local docker daemon store to the target defined in the configuration.
// Docker image cannot be pushed by digest to a registry. So to be able to push the image inside the targeted repository
// the same behaviour than for multi architecture images is used: all the images are tagged for the targeted repository
// and then pushed.
// Every time a new image is pushed under a tag, the previous tagged image will be untagged. But this untagged image
// remains accessible using its digest. So right after pushing it, the image is resolved to grab its digest from the
// registry and can be added to the index.
// The final workflow is then:
//  - tag the image to push with targeted reference
//  - push the image using a docker `ImageAPIClient`
//  - resolve the pushed image to grab its digest
func pushImageToTarget(ctx context.Context, src string, cfg fixupConfig) (ocischemav1.Descriptor, error) {
	taggedRef := reference.TagNameOnly(cfg.targetRef)

	if err := cfg.imageClient.ImageTag(ctx, src, cfg.targetRef.String()); err != nil {
		return ocischemav1.Descriptor{}, fmt.Errorf("failed to push image %q, make sure the image exists locally: %s", src, err)
	}

	if err := pushTaggedImage(ctx, cfg.imageClient, cfg.targetRef, cfg.pushOut); err != nil {
		return ocischemav1.Descriptor{}, fmt.Errorf("failed to push image %q: %s", src, err)
	}

	_, descriptor, err := cfg.resolver.Resolve(ctx, taggedRef.String())
	if err != nil {
		return ocischemav1.Descriptor{}, fmt.Errorf("failed to resolve %q after pushing it: %s", taggedRef, err)
	}

	return descriptor, nil
}
