package remotes

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/docker/cnab-to-oci/converter"
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

	config := converter.CreateBundleConfig(tests.MakeTestBundle())
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
	b, err := Pull(context.Background(), ref, resolver)
	assert.NilError(t, err)
	expectedBundle := tests.MakeTestBundle()
	assert.DeepEqual(t, expectedBundle, b)
}

func ExamplePull() {
	// Use remotes.CreateResolver for creating your remotes.Resolver
	resolver := createExampleResolver()
	ref, err := reference.ParseNamed("my.registry/namespace/my-app:my-tag")
	if err != nil {
		panic(err)
	}

	// Pull the CNAB and get the bundle
	resultBundle, err := Pull(context.Background(), ref, resolver)
	if err != nil {
		panic(err)
	}

	resultBundle.WriteTo(os.Stdout)
	// Output:
	// {
	//     "name": "my-app",
	//     "version": "0.1.0",
	//     "description": "description",
	//     "keywords": [
	//         "keyword1",
	//         "keyword2"
	//     ],
	//     "maintainers": [
	//         {
	//             "name": "docker",
	//             "email": "docker@docker.com",
	//             "url": "docker.com"
	//         }
	//     ],
	//     "invocationImages": [
	//         {
	//             "imageType": "docker",
	//             "image": "my.registry/namespace/my-app@sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
	//             "size": 506,
	//             "mediaType": "application/vnd.docker.distribution.manifest.v2+json"
	//         }
	//     ],
	//     "images": {
	//         "image-1": {
	//             "imageType": "oci",
	//             "image": "my.registry/namespace/my-app@sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
	//             "size": 507,
	//             "mediaType": "application/vnd.oci.image.manifest.v1+json",
	//             "description": "nginx:2.12"
	//         }
	//     },
	//     "actions": {
	//         "action-1": {
	//             "Modifies": true
	//         }
	//     },
	//     "parameters": {
	//         "param1": {
	//             "type": "type",
	//             "defaultValue": "hello",
	//             "allowedValues": [
	//                 "value1",
	//                 true,
	//                 1
	//             ],
	//             "required": false,
	//             "metadata": {},
	//             "destination": {
	//                 "path": "/some/path",
	//                 "env": "env_var"
	//             }
	//         }
	//     },
	//     "credentials": {
	//         "cred-1": {
	//             "path": "/some/path",
	//             "env": "env-var"
	//         }
	//     }
	// }
}

const (
	bufBundleManifest = `{
  "schemaVersion": 1,
  "manifests": [
    {
      "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
      "digest": "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
      "size": 315,
      "annotations": {
        "io.cnab.type": "config"
      }
    },
    {
      "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
      "digest": "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
      "size": 506,
      "annotations": {
        "io.cnab.type": "invocation"
      }
    },
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
      "size": 507,
      "annotations": {
        "io.cnab.component_name": "image-1",
        "io.cnab.original_name": "nginx:2.12",
        "io.cnab.type": "component"
      }
    }
  ],
  "annotations": {
    "io.cnab.keywords": "[\"keyword1\",\"keyword2\"]",
    "io.cnab.runtime_version": "v1.0.0-WD",
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
   "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
   "config": {
      "mediaType": "application/vnd.docker.container.image.v1+json",
      "size": 315,
      "digest": "sha256:e2337974e94637d3fab7004f87501e605b08bca3adf9ecd356909a9329da128a"
   },
   "layers": null
}`

	bufBundleConfig = `{
  "schema_version": "v1.0.0-WD",
  "actions": {
    "action-1": {
      "Modifies": true
    }
  },
  "parameters": {
    "param1": {
      "type": "type",
      "defaultValue": "hello",
      "allowedValues": [
        "value1",
        true,
        1
      ],
      "required": false,
      "metadata": {},
      "destination": {
        "path": "/some/path",
        "env": "env_var"
      }
    }
  },
  "credentials": {
    "cred-1": {
      "path": "/some/path",
      "env": "env-var"
    }
  }
}`
)

func createExampleResolver() *mockResolver {
	buf := []*bytes.Buffer{
		// Bundle index
		bytes.NewBuffer([]byte(bufBundleManifest)),
		// Bundle config manifest
		bytes.NewBuffer([]byte(bundleConfigManifestDescriptor)),
		// Bundle config
		bytes.NewBuffer([]byte(bufBundleConfig)),
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
