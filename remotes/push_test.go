package remotes

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/remotes"
	//"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/cnab-to-oci/test"
	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"gotest.tools/assert"
)

// Mock remote.Resolver interface
type mockResolver struct {
	pushedReferences []string
	pusher           *mockPusher
}

func (r *mockResolver) Resolve(ctx context.Context, ref string) (name string, desc ocischemav1.Descriptor, err error) {
	return "", ocischemav1.Descriptor{}, nil
}
func (r *mockResolver) Fetcher(ctx context.Context, ref string) (remotes.Fetcher, error) {
	return nil, nil
}
func (r *mockResolver) Pusher(ctx context.Context, ref string) (remotes.Pusher, error) {
	r.pushedReferences = append(r.pushedReferences, ref)
	return r.pusher, nil
}

// Mock remotes.Pusher interface
type mockPusher struct {
	pushedDescriptors []ocischemav1.Descriptor
	buffers           []*bytes.Buffer
}

func (p *mockPusher) Push(ctx context.Context, d ocischemav1.Descriptor) (content.Writer, error) {
	p.pushedDescriptors = append(p.pushedDescriptors, d)
	buf := &bytes.Buffer{}
	p.buffers = append(p.buffers, buf)
	return &mockWriter{
		WriteCloser: nopWriteCloser{Buffer: buf},
	}, nil
}

// Mock content.Writer interface
type mockWriter struct {
	io.WriteCloser
}

func (w mockWriter) Digest() digest.Digest { return "" }
func (w mockWriter) Commit(ctx context.Context, size int64, expected digest.Digest, opts ...content.Opt) error {
	return nil
}
func (w mockWriter) Status() (content.Status, error) { return content.Status{}, nil }
func (w mockWriter) Truncate(size int64) error       { return nil }

type nopWriteCloser struct {
	*bytes.Buffer
}

func (n nopWriteCloser) Close() error { return nil }

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
      "mediaType": "application/io.docker.cnab.config.v1.0.0-WD+json",
      "digest": "sha256:e2337974e94637d3fab7004f87501e605b08bca3adf9ecd356909a9329da128a",
      "size": 314
    },
    {
      "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
      "digest": "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
      "size": 250,
      "annotations": {
        "io.cnab.type": "invocation"
      }
    },
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
      "size": 250,
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
	b := test.MakeTestBundle()
	ref, err := reference.ParseNamed("my.registry/namespace/my-app:my-tag")
	assert.NilError(t, err)

	// push the bundle
	_, err = Push(context.Background(), b, ref, resolver)
	assert.NilError(t, err)
	assert.Equal(t, len(resolver.pushedReferences), 2)
	assert.Equal(t, len(pusher.pushedDescriptors), 2)
	assert.Equal(t, len(pusher.buffers), 2)

	// check pushed config
	assert.Equal(t, "my.registry/namespace/my-app", resolver.pushedReferences[0])
	assert.Equal(t, "application/io.docker.cnab.config.v1.0.0-WD+json", pusher.pushedDescriptors[0].MediaType)
	assert.Equal(t, oneLiner(expectedBundleConfig), pusher.buffers[0].String())

	// check pushed bundle manifest index
	assert.Equal(t, "my.registry/namespace/my-app:my-tag", resolver.pushedReferences[1])
	assert.Equal(t, ocischemav1.MediaTypeImageIndex, pusher.pushedDescriptors[1].MediaType)
	assert.Equal(t, oneLiner(expectedBundleManifest), pusher.buffers[1].String())
}

func oneLiner(s string) string {
	return strings.Replace(strings.Replace(s, " ", "", -1), "\n", "", -1)
}
