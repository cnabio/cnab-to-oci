package remotes

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	oci "github.com/docker/cnab-to-oci"
	"github.com/docker/cnab-to-oci/test"
	"github.com/docker/distribution/reference"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"gotest.tools/assert"
)

func TestPull(t *testing.T) {
	index := test.MakeTestOCIIndex()
	bufBundleManifest, err := json.Marshal(index)
	assert.NilError(t, err)

	bundleConfigManifestDescriptor := []byte(`{
   "schemaVersion": 2,
   "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
   "config": {
      "mediaType": "application/vnd.docker.container.image.v1+json",
      "size": 314,
      "digest": "sha256:e2337974e94637d3fab7004f87501e605b08bca3adf9ecd356909a9329da128a"
   },
   "layers": null
}`)

	config := oci.CreateBundleConfig(test.MakeTestBundle())
	bufBundleConfig, err := json.Marshal(config)
	assert.NilError(t, err)

	fetcher := &mockFetcher{indexBuffers: []*bytes.Buffer{
		// Bundle index
		bytes.NewBuffer(bufBundleManifest),
		// Bundle config manifest
		bytes.NewBuffer(bundleConfigManifestDescriptor),
		// Bundle config
		bytes.NewBuffer(bufBundleConfig),
	}}
	resolver := &mockResolver{
		fetcher: fetcher,
		resolvedDescriptors: []ocischemav1.Descriptor{
			// Bundle index descriptor
			ocischemav1.Descriptor{MediaType: ocischemav1.MediaTypeImageIndex},
			// Bundle config manifest descriptor
			ocischemav1.Descriptor{
				MediaType: ocischemav1.MediaTypeDescriptor,
				Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
			},
			// Bundle config descriptor
			ocischemav1.Descriptor{MediaType: ocischemav1.MediaTypeImageIndex},
		},
	}
	ref, err := reference.ParseNamed("my.registry/namespace/my-app:my-tag")
	assert.NilError(t, err)

	// Pull the CNAB and get the bundle
	b, err := Pull(context.Background(), ref, resolver)
	assert.NilError(t, err)
	expectedBundle := test.MakeTestBundle()
	assert.DeepEqual(t, expectedBundle, b)
}
