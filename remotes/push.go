package remotes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/cnab-to-oci/converter"
	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ManifestOption is a callback used to customize a manifest before pushing it
type ManifestOption func(*ocischemav1.Index) error

// Push pushes a bundle as an OCI Image Index manifest
func Push(ctx context.Context, b *bundle.Bundle, ref reference.Named, resolver remotes.Resolver, allowFallbacks bool, options ...ManifestOption) (ocischemav1.Descriptor, error) {
	logrus.Infof("Pushing CNAB Bundle %s", ref)
	bundleConfig, err := converter.CreateBundleConfig(b).PrepareForPush()
	if err != nil {
		return ocischemav1.Descriptor{}, err
	}
	logrus.Infof("Pushing CNAB Bundle Config")
	confManifestDescriptor, err := pushBundleConfig(ctx, resolver, ref.Name(), bundleConfig, allowFallbacks)
	if err != nil {
		return ocischemav1.Descriptor{}, fmt.Errorf("error while pushing bundle config manifest: %s", err)
	}
	logrus.Info("CNAB Bundle Config pushed")

	logrus.Info("Pushing CNAB Index")
	indexDescriptor, indexPayload, err := prepareIndex(b, ref, confManifestDescriptor, options...)
	if err != nil {
		return ocischemav1.Descriptor{}, err
	}
	// Push the bundle index
	logrus.Info("Trying to push OCI Index")
	logrus.Info(string(indexPayload))
	logrus.Info("OCI Index Descriptor")
	logPayload(indexDescriptor)

	if err := pushPayload(ctx, resolver, ref.String(), indexDescriptor, indexPayload); err != nil {
		if !allowFallbacks {
			return ocischemav1.Descriptor{}, err
		}
		// retry with a docker manifestlist
		indexDescriptor, indexPayload, err = prepareIndexNonOCI(b, ref, confManifestDescriptor, options...)
		if err != nil {
			return ocischemav1.Descriptor{}, err
		}
		logrus.Info("Trying to push Index with Manifest list as fallback")
		logrus.Info(string(indexPayload))
		logrus.Info("Manifest list Descriptor")
		logPayload(indexDescriptor)
		if err := pushPayload(ctx, resolver, ref.String(), indexDescriptor, indexPayload); err != nil {
			return ocischemav1.Descriptor{}, err
		}
	}
	logrus.Infof("CNAB Index pushed")
	return indexDescriptor, nil
}

func prepareIndex(b *bundle.Bundle, ref reference.Named, confDescriptor ocischemav1.Descriptor, options ...ManifestOption) (ocischemav1.Descriptor, []byte, error) {
	ix, err := convertIndexAndApplyOptions(b, ref, confDescriptor, options...)
	if err != nil {
		return ocischemav1.Descriptor{}, nil, err
	}
	indexPayload, err := json.Marshal(ix)
	if err != nil {
		return ocischemav1.Descriptor{}, nil, fmt.Errorf("invalid bundle manifest %q: %s", ref, err)
	}
	indexDescriptor := ocischemav1.Descriptor{
		Digest:    digest.FromBytes(indexPayload),
		MediaType: ocischemav1.MediaTypeImageIndex,
		Size:      int64(len(indexPayload)),
	}
	return indexDescriptor, indexPayload, nil
}

type ociIndexWrapper struct {
	ocischemav1.Index
	MediaType string `json:"mediaType,omitempty"`
}

func convertIndexAndApplyOptions(b *bundle.Bundle, ref reference.Named, confDescriptor ocischemav1.Descriptor, options ...ManifestOption) (*ocischemav1.Index, error) {
	ix, err := converter.ConvertBundleToOCIIndex(b, ref, confDescriptor)
	if err != nil {
		return nil, err
	}
	for _, opts := range options {
		if err := opts(ix); err != nil {
			return nil, fmt.Errorf("failed to prepare bundle manifest %q: %s", ref, err)
		}
	}
	return ix, nil
}

func prepareIndexNonOCI(b *bundle.Bundle, ref reference.Named, confDescriptor ocischemav1.Descriptor, options ...ManifestOption) (ocischemav1.Descriptor, []byte, error) {
	ix, err := convertIndexAndApplyOptions(b, ref, confDescriptor, options...)
	if err != nil {
		return ocischemav1.Descriptor{}, nil, err
	}
	w := &ociIndexWrapper{Index: *ix, MediaType: images.MediaTypeDockerSchema2ManifestList}
	w.SchemaVersion = 2
	indexPayload, err := json.Marshal(w)
	if err != nil {
		return ocischemav1.Descriptor{}, nil, fmt.Errorf("invalid bundle manifest %q: %s", ref, err)
	}
	indexDescriptor := ocischemav1.Descriptor{
		Digest:    digest.FromBytes(indexPayload),
		MediaType: images.MediaTypeDockerSchema2ManifestList,
		Size:      int64(len(indexPayload)),
	}
	return indexDescriptor, indexPayload, nil
}

func pushPayload(ctx context.Context, resolver remotes.Resolver, reference string, descriptor ocischemav1.Descriptor, payload []byte) error {
	pusher, err := resolver.Pusher(ctx, reference)
	if err != nil {
		return err
	}
	writer, err := pusher.Push(ctx, descriptor)
	if err != nil {
		if errors.Cause(err) == errdefs.ErrAlreadyExists {
			return nil
		}
		return err
	}
	defer writer.Close()
	if _, err := writer.Write(payload); err != nil {
		if errors.Cause(err) == errdefs.ErrAlreadyExists {
			return nil
		}
		return err
	}
	err = writer.Commit(ctx, descriptor.Size, descriptor.Digest)
	if errors.Cause(err) == errdefs.ErrAlreadyExists {
		return nil
	}
	return err
}

func pushBundleConfig(ctx context.Context, resolver remotes.Resolver, reference string, bundleConfig *converter.PreparedBundleConfig, allowFallbacks bool) (ocischemav1.Descriptor, error) {
	logrus.Info("Trying to push CNAB Bundle Config")
	logrus.Info("CNAB Bundle Config Descriptor")
	logPayload(bundleConfig.ConfigBlobDescriptor)
	if err := pushPayload(ctx, resolver, reference, bundleConfig.ConfigBlobDescriptor, bundleConfig.ConfigBlob); err != nil {
		if allowFallbacks && bundleConfig.Fallback != nil {
			logrus.Infof("Failed to push CNAB Bundle Config, trying with a fallback method")
			return pushBundleConfig(ctx, resolver, reference, bundleConfig.Fallback, allowFallbacks)
		}
		return ocischemav1.Descriptor{}, err
	}
	logrus.Info("Trying to push CNAB Bundle Config Manifest")
	logrus.Info(string(bundleConfig.Manifest))
	logrus.Info("CNAB Bundle Config Manifest Descriptor")
	logPayload(bundleConfig.ManifestDescriptor)
	if err := pushPayload(ctx, resolver, reference, bundleConfig.ManifestDescriptor, bundleConfig.Manifest); err != nil {
		if allowFallbacks && bundleConfig.Fallback != nil {
			logrus.Info("Failed to push CNAB Bundle Config Manifest, trying with a fallback method")
			return pushBundleConfig(ctx, resolver, reference, bundleConfig.Fallback, allowFallbacks)
		}
		return ocischemav1.Descriptor{}, err
	}
	return bundleConfig.ManifestDescriptor, nil
}
