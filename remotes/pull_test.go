package remotes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/docker/cnab-to-oci/tests"
	"github.com/docker/distribution/reference"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"gotest.tools/assert"
)

func TestPull(t *testing.T) {
	index := tests.MakeTestOCIIndex()
	bufBundleManifest, err := json.Marshal(index)
	assert.NilError(t, err)

	bundleConfigManifestDescriptor := []byte(`{
   "schemaVersion": 2,
   "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
   "config": {
      "mediaType": "application/vnd.docker.container.image.v1+json",
      "size": 315,
      "digest": "sha256:e2337974e94637d3fab7004f87501e605b08bca3adf9ecd356909a9329da128a"
   },
   "layers": null
}`)

	b := tests.MakeTestBundle()
	bufBundle, err := json.Marshal(b)
	assert.NilError(t, err)

	fetcher := &mockFetcher{indexBuffers: []*bytes.Buffer{
		// Bundle index
		bytes.NewBuffer(bufBundleManifest),
		// Bundle config manifest
		bytes.NewBuffer(bundleConfigManifestDescriptor),
		// Bundle config
		bytes.NewBuffer(bufBundle),
	}}
	resolver := &mockResolver{
		fetcher: fetcher,
		resolvedDescriptors: []ocischemav1.Descriptor{
			// Bundle index descriptor
			{MediaType: ocischemav1.MediaTypeImageIndex},
			// Bundle config manifest descriptor
			{
				MediaType: ocischemav1.MediaTypeDescriptor,
				Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
			},
			// Bundle config descriptor
			{MediaType: ocischemav1.MediaTypeImageIndex},
		},
	}
	ref, err := reference.ParseNamed("my.registry/namespace/my-app:my-tag")
	assert.NilError(t, err)

	// Pull the CNAB and get the bundle
	b, _, err = Pull(context.Background(), ref, resolver)
	assert.NilError(t, err)
	expectedBundle := tests.MakeTestBundle()
	assert.DeepEqual(t, expectedBundle, b)
}

// nolint: lll
func ExamplePull() {
	// Use remotes.CreateResolver for creating your remotes.Resolver
	resolver := createExampleResolver()
	ref, err := reference.ParseNamed("my.registry/namespace/my-app:my-tag")
	if err != nil {
		panic(err)
	}

	// Pull the CNAB, get the bundle and the associated relocation map
	resultBundle, relocationMap, err := Pull(context.Background(), ref, resolver)
	if err != nil {
		panic(err)
	}

	resultBundle.WriteTo(os.Stdout) //nolint:errcheck
	buf, err := json.Marshal(relocationMap)
	if err != nil {
		panic(err)
	}
	fmt.Printf("\n")
	fmt.Println(string(buf))
	// Output:
	//{"actions":{"action-1":{"modifies":true}},"credentials":{"cred-1":{"env":"env-var","path":"/some/path"}},"custom":{"my-key":"my-value"},"definitions":{"output1Type":{"type":"string"},"param1Type":{"default":"hello","enum":["value1",true,1],"type":["string","boolean","number"]}},"description":"description","images":{"another-image":{"contentDigest":"sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0342","description":"","image":"my.registry/namespace/another-image","imageType":"oci","mediaType":"application/vnd.oci.image.manifest.v1+json","size":507},"image-1":{"contentDigest":"sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341","description":"","image":"my.registry/namespace/image-1","imageType":"oci","mediaType":"application/vnd.oci.image.manifest.v1+json","size":507}},"invocationImages":[{"contentDigest":"sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0343","image":"my.registry/namespace/my-app-invoc","imageType":"docker","mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":506}],"keywords":["keyword1","keyword2"],"maintainers":[{"email":"docker@docker.com","name":"docker","url":"docker.com"}],"name":"my-app","outputs":{"output1":{"applyTo":["install"],"definition":"output1Type","description":"magic","path":"/cnab/app/outputs/magic"}},"parameters":{"param1":{"definition":"param1Type","destination":{"env":"env_var","path":"/some/path"}}},"schemaVersion":"v1.0.0","version":"0.1.0"}
	//{"my.registry/namespace/image-1":"my.registry/namespace/my-app@sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341","my.registry/namespace/my-app-invoc":"my.registry/namespace/my-app@sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341"}
}

const (
	bufBundleManifest = `{
  "schemaVersion": 1,
  "manifests": [
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
      "size": 315,
      "annotations": {
        "io.cnab.manifest.type": "config"
      }
    },
    {
      "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
      "digest": "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
      "size": 506,
      "annotations": {
        "io.cnab.manifest.type": "invocation"
      }
    },
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
      "size": 507,
      "annotations": {
        "io.cnab.component.name": "image-1",
        "io.cnab.manifest.type": "component"
      }
    }
  ],
  "annotations": {
    "io.cnab.keywords": "[\"keyword1\",\"keyword2\"]",
    "io.cnab.runtime_version": "v1.0.0",
    "io.docker.app.format": "cnab",
    "io.docker.type": "app",
    "org.opencontainers.image.authors": "[{\"name\":\"docker\",\"email\":\"docker@docker.com\",\"url\":\"docker.com\"}]",
    "org.opencontainers.image.description": "description",
    "org.opencontainers.image.title": "my-app",
    "org.opencontainers.image.version": "0.1.0"
  }
}`

	bundleConfigManifestDescriptor = `{
   "schemaVersion": 2,
   "config": {
      "mediaType": "application/vnd.cnab.config.v1+json",
      "size": 315,
      "digest": "sha256:e2337974e94637d3fab7004f87501e605b08bca3adf9ecd356909a9329da128a"
   },
   "layers": null
}`
)

func createExampleResolver() *mockResolver {
	b := tests.MakeTestBundle()
	bufBundleConfig, err := json.Marshal(b)
	if err != nil {
		panic(err)
	}
	buf := []*bytes.Buffer{
		// Bundle index
		bytes.NewBuffer([]byte(bufBundleManifest)),
		// Bundle config manifest
		bytes.NewBuffer([]byte(bundleConfigManifestDescriptor)),
		// Bundle config
		bytes.NewBuffer(bufBundleConfig),
	}
	fetcher := &mockFetcher{indexBuffers: buf}
	pusher := &mockPusher{}
	return &mockResolver{
		pusher:  pusher,
		fetcher: fetcher,
		resolvedDescriptors: []ocischemav1.Descriptor{
			// Bundle index descriptor
			{
				MediaType: ocischemav1.MediaTypeImageIndex,
				Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
				Size:      int64(len(bufBundleManifest)),
			},
			// Bundle config manifest descriptor
			{
				MediaType: ocischemav1.MediaTypeDescriptor,
				Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
				Size:      int64(len(bundleConfigManifestDescriptor)),
			},
			// Bundle config descriptor
			{
				MediaType: ocischemav1.MediaTypeImageConfig,
				Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
				Size:      int64(len(bufBundleConfig)),
			},
		},
	}
}
