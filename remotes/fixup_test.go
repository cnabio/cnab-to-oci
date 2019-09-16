package remotes

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"testing"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/platforms"
	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/cnab-to-oci/relocation"
	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"gotest.tools/assert"
)

func TestFixupBundleWithAutoUpdate(t *testing.T) {
	index := ocischemav1.Manifest{}
	bufManifest, err := json.Marshal(index)
	assert.NilError(t, err)
	fetcher := &mockFetcher{indexBuffers: []*bytes.Buffer{
		// Manifest index
		bytes.NewBuffer(bufManifest),
	}}
	pusher := &mockPusher{}
	resolver := &mockResolver{
		pusher:  pusher,
		fetcher: fetcher,
		resolvedDescriptors: []ocischemav1.Descriptor{
			// Resolving source Invocation image manifest descriptor my.registry/namespace/my-app-invoc
			{
				MediaType: ocischemav1.MediaTypeImageManifest,
				Size:      42,
				Digest:    "sha256:beef1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0343",
			},
			// Target Invocation image manifest descriptor my.registry/namespace/my-app@sha256:beef1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0343 for mounting
			{},
			// Resolving source service image manifest descriptor my.registry/namespace/my-service
			{
				MediaType: ocischemav1.MediaTypeImageManifest,
				Size:      43,
				Digest:    "sha256:beef1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0344",
			},
			// Target service image manifest descriptor my.registry/namespace/my-app@sha256:beef1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0344 for mounting
			{},
		},
	}
	b := &bundle.Bundle{
		SchemaVersion: "v1.0.0-WD",
		InvocationImages: []bundle.InvocationImage{
			{
				BaseImage: bundle.BaseImage{
					Image:     "my.registry/namespace/my-app-invoc",
					ImageType: "docker",
				},
			},
		},
		Images: map[string]bundle.Image{
			"my-service": {
				BaseImage: bundle.BaseImage{
					Image:     "my.registry/namespace/my-service",
					ImageType: "docker",
				},
			},
		},
		Name:    "my-app",
		Version: "0.1.0",
	}
	ref, err := reference.ParseNamed("my.registry/namespace/my-app")
	assert.NilError(t, err)
	_, err = FixupBundle(context.TODO(), b, ref, resolver, WithAutoBundleUpdate())
	assert.NilError(t, err)
	expectedBundle := &bundle.Bundle{
		SchemaVersion: "v1.0.0-WD",
		InvocationImages: []bundle.InvocationImage{
			{
				BaseImage: bundle.BaseImage{
					Image:     "my.registry/namespace/my-app-invoc",
					ImageType: "docker",
					MediaType: ocischemav1.MediaTypeImageManifest,
					Size:      42,
					Digest:    "sha256:beef1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0343",
				},
			},
		},
		Images: map[string]bundle.Image{
			"my-service": {
				BaseImage: bundle.BaseImage{
					Image:     "my.registry/namespace/my-service",
					ImageType: "docker",
					MediaType: ocischemav1.MediaTypeImageManifest,
					Size:      43,
					Digest:    "sha256:beef1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0344",
				},
			},
		},
		Name:    "my-app",
		Version: "0.1.0",
	}
	assert.DeepEqual(t, b, expectedBundle)
}

func TestFixupBundleFailsWithDifferentDigests(t *testing.T) {
	index := ocischemav1.Manifest{}
	bufManifest, err := json.Marshal(index)
	assert.NilError(t, err)
	fetcher := &mockFetcher{indexBuffers: []*bytes.Buffer{
		// Manifest index
		bytes.NewBuffer(bufManifest),
	}}
	pusher := &mockPusher{}
	resolver := &mockResolver{
		pusher:  pusher,
		fetcher: fetcher,
		resolvedDescriptors: []ocischemav1.Descriptor{
			// Invocation image manifest descriptor
			{
				MediaType: ocischemav1.MediaTypeImageManifest,
				Size:      42,
				Digest:    "sha256:c0ffeea7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0343",
			},
			{},
		},
	}
	b := &bundle.Bundle{
		SchemaVersion: "v1.0.0-WD",
		InvocationImages: []bundle.InvocationImage{
			{
				BaseImage: bundle.BaseImage{
					Image:     "my.registry/namespace/my-app-invoc",
					ImageType: "docker",
					Digest:    "beef00a7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0343",
					Size:      42,
					MediaType: ocischemav1.MediaTypeImageManifest,
				},
			},
		},
		Name:    "my-app",
		Version: "0.1.0",
	}
	ref, err := reference.ParseNamed("my.registry/namespace/my-app")
	assert.NilError(t, err)
	_, err = FixupBundle(context.TODO(), b, ref, resolver)
	assert.ErrorContains(t, err, "digest differs")
}

func TestFixupBundleFailsWithDifferentSizes(t *testing.T) {
	index := ocischemav1.Manifest{}
	bufManifest, err := json.Marshal(index)
	assert.NilError(t, err)
	fetcher := &mockFetcher{indexBuffers: []*bytes.Buffer{
		// Manifest index
		bytes.NewBuffer(bufManifest),
	}}
	pusher := &mockPusher{}
	resolver := &mockResolver{
		pusher:  pusher,
		fetcher: fetcher,
		resolvedDescriptors: []ocischemav1.Descriptor{
			// Invocation image manifest descriptor
			{
				MediaType: ocischemav1.MediaTypeImageManifest,
				Size:      43,
				Digest:    "sha256:c0ffeea7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0343",
			},
			{},
		},
	}

	b := &bundle.Bundle{
		SchemaVersion: "v1.0.0-WD",
		InvocationImages: []bundle.InvocationImage{
			{
				BaseImage: bundle.BaseImage{
					Image:     "my.registry/namespace/my-app-invoc",
					ImageType: "docker",
					Digest:    "sha256:c0ffeea7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0343",
					Size:      42,
					MediaType: ocischemav1.MediaTypeImageManifest,
				},
			},
		},
		Name:    "my-app",
		Version: "0.1.0",
	}
	ref, err := reference.ParseNamed("my.registry/namespace/my-app")
	assert.NilError(t, err)
	_, err = FixupBundle(context.TODO(), b, ref, resolver)
	assert.ErrorContains(t, err, "size differs")
}

func TestFixupBundleFailsWithDifferentMediaTypes(t *testing.T) {
	index := ocischemav1.Manifest{}
	bufManifest, err := json.Marshal(index)
	assert.NilError(t, err)
	fetcher := &mockFetcher{indexBuffers: []*bytes.Buffer{
		// Manifest index
		bytes.NewBuffer(bufManifest),
	}}
	pusher := &mockPusher{}
	resolver := &mockResolver{
		pusher:  pusher,
		fetcher: fetcher,
		resolvedDescriptors: []ocischemav1.Descriptor{
			// Invocation image manifest descriptor
			{
				MediaType: ocischemav1.MediaTypeImageIndex,
				Size:      42,
				Digest:    "sha256:c0ffeea7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0343",
			},
			{},
		},
	}

	b := &bundle.Bundle{
		SchemaVersion: "v1.0.0-WD",
		InvocationImages: []bundle.InvocationImage{
			{
				BaseImage: bundle.BaseImage{
					Image:     "my.registry/namespace/my-app-invoc",
					ImageType: "docker",
					Digest:    "sha256:c0ffeea7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0343",
					Size:      42,
					MediaType: ocischemav1.MediaTypeImageManifest,
				},
			},
		},
		Name:    "my-app",
		Version: "0.1.0",
	}
	ref, err := reference.ParseNamed("my.registry/namespace/my-app")
	assert.NilError(t, err)
	_, err = FixupBundle(context.TODO(), b, ref, resolver)
	assert.ErrorContains(t, err, "media type differs")
}

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
			var filter platforms.Matcher
			if c.platform != "" {
				filter = platforms.NewMatcher(platforms.MustParse(c.platform))
			}
			assert.NilError(t, fixupPlatforms(context.Background(), &bundle.BaseImage{}, relocation.ImageRelocationMap{}, &imageFixupInfo{
				resolvedDescriptor: ocischemav1.Descriptor{
					MediaType: c.mediaType,
				},
			}, nil, filter))
		})
	}
}

func TestFixupPlatforms(t *testing.T) {
	cases := []struct {
		name           string
		manifest       *testManifest
		filter         []string
		expectedResult *testManifest
		expectedError  string
	}{
		{
			name:           "single-filter",
			manifest:       newTestManifest("linux/amd64", "windows/amd64"),
			filter:         []string{"linux/amd64"},
			expectedResult: newTestManifest("linux/amd64"),
		},
		{
			name:           "multi-filter",
			manifest:       newTestManifest("linux/amd64", "windows/amd64", "linux/arm64"),
			filter:         []string{"linux/amd64", "linux/arm64"},
			expectedResult: newTestManifest("linux/amd64", "linux/arm64"),
		},

		{
			name:          "no-match",
			manifest:      newTestManifest("linux/amd64", "windows/amd64"),
			filter:        []string{"linux/arm64"},
			expectedError: `no descriptor matching the platform filter found in "docker.io/docker/test@sha256:4ff4130e3c087b3dd1ce3d7e9d29316e707c0a793783aa76380a14c1dba9b536"`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// parse filter
			plats, err := toPlatforms(c.filter)
			assert.NilError(t, err)
			filter := platforms.Any(plats...)

			// setup fixupinfo, baseImage
			sourceBytes, err := json.Marshal(c.manifest)
			assert.NilError(t, err)
			sourceDigest := digest.FromBytes(sourceBytes)
			sourceRepo, err := reference.ParseNormalizedNamed("docker/test")
			assert.NilError(t, err)
			targetRepo, err := reference.ParseNormalizedNamed("docker/target")
			assert.NilError(t, err)
			sourceRef, err := reference.WithDigest(sourceRepo, sourceDigest)
			assert.NilError(t, err)
			bi := bundle.BaseImage{
				Image: sourceRef.String(),
			}
			fixupInfo := &imageFixupInfo{
				resolvedDescriptor: ocischemav1.Descriptor{
					Digest:    sourceDigest,
					Size:      int64(len(sourceBytes)),
					MediaType: ocischemav1.MediaTypeImageIndex,
				},
				targetRepo: targetRepo,
				sourceRef:  sourceRef,
			}

			// setup source fetcher
			sourceFetcher := newSourceFetcherWithLocalData(bytesFetcher(sourceBytes))

			// fixup
			err = fixupPlatforms(context.Background(), &bi, relocation.ImageRelocationMap{}, fixupInfo, sourceFetcher, filter)
			if c.expectedError != "" {
				assert.ErrorContains(t, err, c.expectedError)
				return
			}
			assert.NilError(t, err)

			// baseImage.Image should have changed
			// assert.Check(t, bi.Image != sourceRef.String())
			// resolved digest should have changed
			assert.Check(t, fixupInfo.resolvedDescriptor.Digest != sourceDigest)

			// parsing back the resolved manifest and making sure extra fields are still there
			resolvedReader, err := sourceFetcher.Fetch(context.Background(), fixupInfo.resolvedDescriptor)
			assert.NilError(t, err)
			defer resolvedReader.Close()
			resolvedBytes, err := ioutil.ReadAll(resolvedReader)
			assert.NilError(t, err)
			var resolvedManifest testManifest
			assert.NilError(t, json.Unmarshal(resolvedBytes, &resolvedManifest))
			assert.DeepEqual(t, &resolvedManifest, c.expectedResult)
		})
	}
}

type testManifest struct {
	Manifests []testDescriptor `json:"manifests"`
	Foo       string           `json:"foo"`
}

type testDescriptor struct {
	Platform *ocischemav1.Platform `json:"platform,omitempty"`
	Bar      string                `json:"bar"`
}

func newTestManifest(plats ...string) *testManifest {
	m := &testManifest{
		Foo: "bar",
	}
	for _, p := range plats {
		plat := platforms.MustParse(p)
		m.Manifests = append(m.Manifests, testDescriptor{
			Bar:      "baz",
			Platform: &plat,
		})
	}
	return m
}

type bytesFetcher []byte

func (f bytesFetcher) Fetch(_ context.Context, _ ocischemav1.Descriptor) (io.ReadCloser, error) {
	reader := bytes.NewReader(f)
	return ioutil.NopCloser(reader), nil
}
