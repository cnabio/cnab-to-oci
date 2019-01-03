package remotes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/deislabs/duffle/pkg/bundle"

	"github.com/docker/cnab-to-oci/tests"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"gotest.tools/assert"
)

const (
	expectedBundleConfig = `{
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

	expectedBundleManifest = `{
  "schemaVersion": 1,
  "manifests": [
    {
      "mediaType":"application/vnd.docker.distribution.manifest.v2+json",
      "digest":"sha256:f4807c40c28978674e881cb7808fa88d648b1cd19832a78238fc9e7d17443c03",
      "size":315,
      "annotations":{
        "io.cnab.type":"config"
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
)

func TestPush(t *testing.T) {
	pusher := &mockPusher{}
	resolver := &mockResolver{pusher: pusher}
	b := tests.MakeTestBundle()
	ref, err := reference.ParseNamed("my.registry/namespace/my-app:my-tag")
	assert.NilError(t, err)

	// push the bundle
	_, err = Push(context.Background(), b, ref, resolver)
	assert.NilError(t, err)
	assert.Equal(t, len(resolver.pushedReferences), 3)
	assert.Equal(t, len(pusher.pushedDescriptors), 3)
	assert.Equal(t, len(pusher.buffers), 3)

	// check pushed config
	assert.Equal(t, "my.registry/namespace/my-app", resolver.pushedReferences[0])
	assert.Equal(t, schema2.MediaTypeImageConfig, pusher.pushedDescriptors[0].MediaType)
	assert.Equal(t, oneLiner(expectedBundleConfig), pusher.buffers[0].String())

	// check pushed config manifest
	assert.Equal(t, "my.registry/namespace/my-app", resolver.pushedReferences[1])
	assert.Equal(t, schema2.MediaTypeManifest, pusher.pushedDescriptors[1].MediaType)

	// check pushed bundle manifest index
	assert.Equal(t, "my.registry/namespace/my-app:my-tag", resolver.pushedReferences[2])
	assert.Equal(t, ocischemav1.MediaTypeImageIndex, pusher.pushedDescriptors[2].MediaType)
	assert.Equal(t, oneLiner(expectedBundleManifest), pusher.buffers[2].String())
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
	descriptor, err := Push(context.Background(), b, ref, resolver)
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
	//   "digest": "sha256:780346d71e4a7ae4375e1b8c2dfced63ed6429d411667c4c0aa30e0a68b75c1e",
	//   "size": 1121
	// }
}

func createExampleBundle() *bundle.Bundle {
	return tests.MakeTestBundle()
}
