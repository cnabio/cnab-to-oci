package remotes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/cnab-to-oci/converter"
	"github.com/docker/cnab-to-oci/tests"
	"github.com/docker/distribution/reference"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"gotest.tools/assert"
)

const (
	expectedBundleConfig = `{
  "schemaVersion": "v1.0.0-WD",
  "actions": {
    "action-1": {
      "modifies": true
    }
  },
  "definitions": {
    "output1Type": {
      "type": "string"
    },
    "param1Type": {
     "default": "hello",
      "enum": [
          "value1",
          true,
          1
      ],
      "type": [
          "string",
          "boolean",
          "number"
      ]
    }
  },
  "parameters": {
    "param1": {
      "definition": "param1Type",
      "destination": {
        "path": "/some/path",
        "env": "env_var"
      }
    }
  },
  "outputs": {
    "output1": {
      "definition": "output1Type",
      "applyTo": [
        "install"
      ],
      "description": "magic",
      "path": "/cnab/app/outputs/magic"
    }
  },
  "credentials": {
    "cred-1": {
      "path": "/some/path",
      "env": "env-var"
    }
  },
  "custom": {
    "my-key": "my-value"
  }
}`

	expectedBundleManifest = `{
  "schemaVersion": 2,
  "manifests": [
    {
      "mediaType":"application/vnd.oci.image.manifest.v1+json",
      "digest":"sha256:75b3dd7d430a5c5f20908dcb63099adedd555850735dbae833ab3312c6e42208",
      "size":188,
      "annotations":{
        "io.cnab.manifest.type":"config"
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
        "io.cnab.component.name": "another-image",
        "io.cnab.manifest.type": "component"
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
    "io.cnab.runtime_version": "v1.0.0-WD",
    "org.opencontainers.artifactType": "application/vnd.cnab.manifest.v1",
    "org.opencontainers.image.authors": "[{\"name\":\"docker\",\"email\":\"docker@docker.com\",\"url\":\"docker.com\"}]",
    "org.opencontainers.image.description": "description",
    "org.opencontainers.image.title": "my-app",
    "org.opencontainers.image.version": "0.1.0"
  }
}`
	expectedConfigManifest = `{
   "schemaVersion": 2,
   "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
   "config": {
      "mediaType": "application/vnd.docker.container.image.v1+json",
      "size": 539,
      "digest": "sha256:583d58ecba680e28cd4f55fa673d377915259dfb5a5a09b79f4196e53517495f"
   },
   "layers": [
      {
         "mediaType": "application/vnd.docker.container.image.v1+json",
         "size": 539,
         "digest": "sha256:583d58ecba680e28cd4f55fa673d377915259dfb5a5a09b79f4196e53517495f"
      }
   ]
}`
)

func TestPush(t *testing.T) {
	pusher := &mockPusher{}
	resolver := &mockResolver{pusher: pusher}
	b := tests.MakeTestBundle()
	ref, err := reference.ParseNamed("my.registry/namespace/my-app:my-tag")
	assert.NilError(t, err)

	// push the bundle
	_, err = Push(context.Background(), b, ref, resolver, true)
	assert.NilError(t, err)
	assert.Equal(t, len(resolver.pushedReferences), 3)
	assert.Equal(t, len(pusher.pushedDescriptors), 3)
	assert.Equal(t, len(pusher.buffers), 3)

	// check pushed config
	assert.Equal(t, "my.registry/namespace/my-app", resolver.pushedReferences[0])
	assert.Equal(t, converter.CNABConfigMediaType, pusher.pushedDescriptors[0].MediaType)
	assert.Equal(t, oneLiner(expectedBundleConfig), pusher.buffers[0].String())

	// check pushed config manifest
	assert.Equal(t, "my.registry/namespace/my-app", resolver.pushedReferences[1])
	assert.Equal(t, ocischemav1.MediaTypeImageManifest, pusher.pushedDescriptors[1].MediaType)

	// check pushed bundle manifest index
	assert.Equal(t, "my.registry/namespace/my-app:my-tag", resolver.pushedReferences[2])
	assert.Equal(t, ocischemav1.MediaTypeImageIndex, pusher.pushedDescriptors[2].MediaType)
	assert.Equal(t, oneLiner(expectedBundleManifest), pusher.buffers[2].String())
}

func TestFallbackConfigManifest(t *testing.T) {
	// Make the pusher return an error for the first two calls
	// so that the fallbacks kick in and we get the non-oci
	// config manifest.
	pusher := newMockPusher([]error{
		errors.New("1"),
		errors.New("2"),
		nil,
		nil,
		nil,
		nil,
		nil})
	resolver := &mockResolver{pusher: pusher}
	b := tests.MakeTestBundle()
	ref, err := reference.ParseNamed("my.registry/namespace/my-app:my-tag")
	assert.NilError(t, err)

	// push the bundle
	_, err = Push(context.Background(), b, ref, resolver, true)
	assert.NilError(t, err)
	assert.Equal(t, expectedConfigManifest, pusher.buffers[3].String())
}

func oneLiner(s string) string {
	return strings.Replace(strings.Replace(s, " ", "", -1), "\n", "", -1)
}

func ExamplePush() {
	resolver := createExampleResolver()
	b := createExampleBundle()
	ref, err := reference.ParseNamed("my.registry/namespace/my-app:my-tag")
	if err != nil {
		panic(err)
	}

	// Push the bundle here
	descriptor, err := Push(context.Background(), b, ref, resolver, true)
	if err != nil {
		panic(err)
	}

	bytes, err := json.MarshalIndent(descriptor, "", "  ")
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s", string(bytes))

	// Output:
	// {
	//   "mediaType": "application/vnd.oci.image.index.v1+json",
	//   "digest": "sha256:ad9bf48bfc84342aae1017a486722b7b22c82a5f31bb2c4f6da81255e5aa09b5",
	//   "size": 1363
	// }
}

func createExampleBundle() *bundle.Bundle {
	return tests.MakeTestBundle()
}
