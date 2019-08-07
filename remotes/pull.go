package remotes

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/cli/opts"
	"github.com/docker/cnab-to-oci/converter"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client/auth"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

// Pull pulls a bundle from an OCI Image Index manifest
func Pull(ctx context.Context, ref reference.Named, resolver remotes.Resolver) (*bundle.Bundle, error) {
	logrus.Infof("Pulling CNAB Bundle %s", ref)
	index, err := getIndex(ctx, ref, resolver)
	if err != nil {
		return nil, err
	}
	config, err := getConfig(ctx, ref, resolver, index)
	if err != nil {
		return nil, err
	}
	return converter.ConvertOCIIndexToBundle(&index, &config, ref)
}

func getIndex(ctx context.Context, ref auth.Scope, resolver remotes.Resolver) (ocischemav1.Index, error) {
	logrus.Info("Getting OCI Index Descriptor")
	resolvedRef, indexDescriptor, err := resolver.Resolve(ctx, ref.String())
	if err != nil {
		return ocischemav1.Index{}, fmt.Errorf("failed to resolve bundle manifest %q: %s", ref, err)
	}
	if indexDescriptor.MediaType != ocischemav1.MediaTypeImageIndex && indexDescriptor.MediaType != images.MediaTypeDockerSchema2ManifestList {
		return ocischemav1.Index{}, fmt.Errorf("invalid media type %q for bundle manifest", indexDescriptor.MediaType)
	}
	logPayload(indexDescriptor)

	logrus.Infof("Fetching OCI Index %s", indexDescriptor.Digest)
	indexPayload, err := pullPayload(ctx, resolver, resolvedRef, indexDescriptor)
	if err != nil {
		return ocischemav1.Index{}, fmt.Errorf("failed to pull bundle manifest %q: %s", ref, err)
	}
	var index ocischemav1.Index
	if err := json.Unmarshal(indexPayload, &index); err != nil {
		return ocischemav1.Index{}, fmt.Errorf("failed to pull bundle manifest %q: %s", ref, err)
	}
	logPayload(index)

	return index, nil
}

func getConfig(ctx context.Context, ref opts.NamedOption, resolver remotes.Resolver, index ocischemav1.Index) (converter.BundleConfig, error) {
	logrus.Info("Getting Bundle Config Manifest Descriptor")
	// config is wrapped in an image manifest. So we first pull the manifest
	// and then the config blob within it
	configManifestDescriptor, err := converter.GetBundleConfigManifestDescriptor(&index)
	if err != nil {
		return converter.BundleConfig{}, fmt.Errorf("failed to get bundle config manifest from %q: %s", ref, err)
	}
	repoOnly, err := reference.ParseNormalizedNamed(ref.Name())
	if err != nil {
		return converter.BundleConfig{}, fmt.Errorf("invalid bundle config manifest reference name %q: %s", ref, err)
	}
	logPayload(configManifestDescriptor)

	logrus.Infof("Getting Bundle Config Manifest %s", configManifestDescriptor.Digest)
	configManifestRef, err := reference.WithDigest(repoOnly, configManifestDescriptor.Digest)
	if err != nil {
		return converter.BundleConfig{}, fmt.Errorf("invalid bundle config manifest reference name %q: %s", ref, err)
	}
	configManifestPayload, err := pullPayload(ctx, resolver, configManifestRef.String(), configManifestDescriptor)
	if err != nil {
		return converter.BundleConfig{}, fmt.Errorf("failed to pull bundle config manifest %q: %s", ref, err)
	}
	var manifest ocischemav1.Manifest
	if err := json.Unmarshal(configManifestPayload, &manifest); err != nil {
		return converter.BundleConfig{}, err
	}
	logPayload(manifest)

	// Pull now the config itself
	logrus.Infof("Fetching Bundle Config %s", manifest.Config.Digest)
	configRef, err := reference.WithDigest(repoOnly, manifest.Config.Digest)
	if err != nil {
		return converter.BundleConfig{}, fmt.Errorf("invalid bundle config reference name %q: %s", ref, err)
	}
	configPayload, err := pullPayload(ctx, resolver, configRef.String(), ocischemav1.Descriptor{
		Digest:    manifest.Config.Digest,
		MediaType: manifest.Config.MediaType,
		Size:      manifest.Config.Size,
	})
	if err != nil {
		return converter.BundleConfig{}, fmt.Errorf("failed to pull bundle config %q: %s", ref, err)
	}
	var config converter.BundleConfig
	if err := json.Unmarshal(configPayload, &config); err != nil {
		return converter.BundleConfig{}, fmt.Errorf("failed to pull bundle config %q: %s", ref, err)
	}
	logPayload(config)

	return config, nil
}

func pullPayload(ctx context.Context, resolver remotes.Resolver, reference string, descriptor ocischemav1.Descriptor) ([]byte, error) {
	fetcher, err := resolver.Fetcher(ctx, reference)
	if err != nil {
		return nil, err
	}
	reader, err := fetcher.Fetch(ctx, descriptor)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return ioutil.ReadAll(reader)
}
