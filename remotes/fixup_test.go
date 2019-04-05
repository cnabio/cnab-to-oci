package remotes

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"testing"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
	ocischema "github.com/opencontainers/image-spec/specs-go"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"gotest.tools/assert"
)

func TestFixupPlatformShortPaths(t *testing.T) {
	// those cases should not need to fetch any data
	cases := []struct {
		name      string
		platform  string
		mediaType string
	}{
		{
			name:      "no-filter",
			mediaType: ocischemav1.MediaTypeImageIndex,
		},
		{
			name:      "oci-image",
			platform:  "linux/amd64",
			mediaType: ocischemav1.MediaTypeImageManifest,
		},
		{
			name:      "docker-image",
			platform:  "linux/amd64",
			mediaType: images.MediaTypeDockerSchema2Manifest,
		},
		{
			name:      "docker-image-schema1",
			platform:  "linux/amd64",
			mediaType: images.MediaTypeDockerSchema1Manifest,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.NilError(t, fixupPlatform(context.Background(), fixupConfig{platform: c.platform}, nil, &imageFixupInfo{
				resolvedDescriptor: ocischemav1.Descriptor{
					MediaType: c.mediaType,
				},
			}, nil))
		})
	}
}

type fetchSetup struct {
	descriptor ocischemav1.Descriptor
	fetcher    remotes.Fetcher
}

type bytesFetcher []byte

func (f bytesFetcher) Fetch(_ context.Context, _ ocischemav1.Descriptor) (io.ReadCloser, error) {
	reader := bytes.NewReader(f)
	return ioutil.NopCloser(reader), nil
}

func createFetchSetup(t *testing.T, ociFormat bool, descriptors ...ocischemav1.Descriptor) fetchSetup {
	t.Helper()
	var (
		rootManifest []byte
		mediaType    string
	)
	if ociFormat {
		m := ocischemav1.Index{
			Versioned: ocischema.Versioned{SchemaVersion: 2},
			Manifests: descriptors,
		}
		bytes, err := json.Marshal(&m)
		assert.NilError(t, err)
		rootManifest = bytes
		mediaType = ocischemav1.MediaTypeImageIndex
	} else {
		dockerDescriptors := make([]manifestlist.ManifestDescriptor, len(descriptors))
		for ix, descriptor := range descriptors {
			dockerDesc := manifestlist.ManifestDescriptor{
				Descriptor: distribution.Descriptor{
					MediaType: descriptor.MediaType,
					Size:      descriptor.Size,
					Digest:    descriptor.Digest,
				},
			}
			if descriptor.Platform != nil {
				dockerDesc.Platform = manifestlist.PlatformSpec{
					Architecture: descriptor.Platform.Architecture,
					OS:           descriptor.Platform.OS,
				}
			}
			dockerDescriptors[ix] = dockerDesc
		}
		m, err := manifestlist.FromDescriptors(dockerDescriptors)
		assert.NilError(t, err)
		bytes, err := m.MarshalJSON()
		assert.NilError(t, err)
		rootManifest = bytes
		mediaType = images.MediaTypeDockerSchema2ManifestList
	}

	rootDescriptor := ocischemav1.Descriptor{
		Digest:    digest.FromBytes(rootManifest),
		MediaType: mediaType,
		Size:      int64(len(rootManifest)),
	}
	return fetchSetup{
		descriptor: rootDescriptor,
		fetcher:    bytesFetcher(rootManifest),
	}
}

func TestFixupPlatformMultiArch(t *testing.T) {
	linuxAmd64Descriptor := ocischemav1.Descriptor{
		Digest:    digest.FromString("linux/amd64"),
		MediaType: ocischemav1.MediaTypeImageManifest,
		Platform: &ocischemav1.Platform{
			Architecture: "amd64",
			OS:           "linux",
		},
	}
	linuxArm64Descriptor := ocischemav1.Descriptor{
		Digest:    digest.FromString("linux/arm64"),
		MediaType: ocischemav1.MediaTypeImageManifest,
		Platform: &ocischemav1.Platform{
			Architecture: "arm64",
			OS:           "linux",
		},
	}
	noPlatDescriptor := ocischemav1.Descriptor{
		Digest:    digest.FromString("noplat"),
		MediaType: ocischemav1.MediaTypeImageManifest,
	}
	cases := []struct {
		name               string
		platform           string
		fetchSetup         fetchSetup
		expectedDescriptor ocischemav1.Descriptor
		expectedError      string
	}{
		{
			name:               "match-ociindex",
			platform:           "linux/amd64",
			fetchSetup:         createFetchSetup(t, true, noPlatDescriptor, linuxArm64Descriptor, linuxAmd64Descriptor),
			expectedDescriptor: linuxAmd64Descriptor,
		},
		{
			name:               "match-manifestlist",
			platform:           "linux/amd64",
			fetchSetup:         createFetchSetup(t, false, noPlatDescriptor, linuxArm64Descriptor, linuxAmd64Descriptor),
			expectedDescriptor: linuxAmd64Descriptor,
		},
		{
			name:          "no-match-ociindex",
			platform:      "windows/amd64",
			fetchSetup:    createFetchSetup(t, true, noPlatDescriptor, linuxArm64Descriptor, linuxAmd64Descriptor),
			expectedError: `no image found for platform "windows/amd64" in "docker.io/library/somerepo@sha256:51ae099fc17b7c56dc5e0b0a410cfa48928995776fa4b419cbfd4dca5605a85b"`,
		},
		{
			name:          "no-match-manifestlist",
			platform:      "windows/amd64",
			fetchSetup:    createFetchSetup(t, false, noPlatDescriptor, linuxArm64Descriptor, linuxAmd64Descriptor),
			expectedError: `no image found for platform "windows/amd64" in "docker.io/library/somerepo@sha256:8cc1363b2109e9c3e7b9b6e728669eb42eb25b09f8351d87236dda7cc26c6891"`,
		},
	}
	targetRepo, err := reference.ParseNormalizedNamed("somerepo")
	assert.NilError(t, err)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sourceRef, err := reference.WithDigest(targetRepo, c.fetchSetup.descriptor.Digest)
			assert.NilError(t, err)
			baseImage := &bundle.BaseImage{}
			fixupInfo := &imageFixupInfo{resolvedDescriptor: c.fetchSetup.descriptor, targetRepo: targetRepo, sourceRef: sourceRef}
			err = fixupPlatform(context.Background(), fixupConfig{platform: c.platform}, baseImage, fixupInfo, c.fetchSetup.fetcher)
			if c.expectedError != "" {
				assert.ErrorContains(t, err, c.expectedError)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, fixupInfo.resolvedDescriptor, c.expectedDescriptor)
			expectedImage, err := reference.WithDigest(targetRepo, c.expectedDescriptor.Digest)
			assert.NilError(t, err)
			assert.Equal(t, expectedImage.String(), baseImage.Image)
		})
	}
}
