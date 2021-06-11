package remotes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/docker/distribution/reference"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"gotest.tools/v3/assert"
)

type twoStepReader struct {
	first        []byte
	second       []byte
	hasReadFirst bool
}

func (r *twoStepReader) Read(p []byte) (n int, err error) {
	if r.hasReadFirst {
		return copy(p, r.second), nil
	}
	r.hasReadFirst = true
	return copy(p, r.first), nil
}
func (r *twoStepReader) Close() error {
	return nil
}

func TestRemoteReaderAtShortReads(t *testing.T) {
	helloWorld := []byte("Hello world!")
	r := &twoStepReader{
		first:  helloWorld[:5],
		second: helloWorld[5:],
	}
	tested := &remoteReaderAt{
		ReadCloser: r,
		size:       int64(len(helloWorld)),
	}

	actual := make([]byte, len(helloWorld))
	n, err := tested.ReadAt(actual, 0)
	assert.NilError(t, err)
	assert.Equal(t, n, len(helloWorld))
	assert.DeepEqual(t, helloWorld, actual)
}

func TestMountOnPush(t *testing.T) {
	hasMounted := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Testing if mount is called, mount API call is in the form of:
		// POST http://<REGISTRY>/<REPO>/<IMAGE>/blobs/uploads?from=<REPO2>/<IMAGE2>
		if strings.Contains(r.URL.EscapedPath(), "library/test/blobs/uploads/") && strings.Contains(r.URL.Query().Get("from"), "library/busybox") {
			hasMounted = true
		}
		// We don't really care what we send here.
		w.WriteHeader(404)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	resolver := docker.NewResolver(docker.ResolverOptions{
		PlainHTTP: true,
	})

	u, err := url.Parse(server.URL)
	assert.NilError(t, err)

	r, err := resolver.Pusher(context.TODO(), u.Hostname()+":"+u.Port()+"/library/test")
	assert.NilError(t, err)

	ref, err := reference.WithName(u.Hostname() + "/library/busybox")
	assert.NilError(t, err)

	desc := ocischemav1.Descriptor{}
	_, _ = pushWithAnnotation(context.TODO(), r, ref, desc)
	assert.Equal(t, hasMounted, true)
}
