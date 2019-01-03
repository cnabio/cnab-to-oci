package remotes

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/remotes"
	"github.com/opencontainers/go-digest"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Mock remote.Resolver interface
type mockResolver struct {
	resolvedDescriptors []ocischemav1.Descriptor
	pushedReferences    []string
	pusher              *mockPusher
	fetcher             *mockFetcher
}

func (r *mockResolver) Resolve(_ context.Context, ref string) (string, ocischemav1.Descriptor, error) {
	descriptor := r.resolvedDescriptors[0]
	r.resolvedDescriptors = r.resolvedDescriptors[1:]
	return ref, descriptor, nil
}
func (r *mockResolver) Fetcher(_ context.Context, ref string) (remotes.Fetcher, error) {
	return r.fetcher, nil
}
func (r *mockResolver) Pusher(_ context.Context, ref string) (remotes.Pusher, error) {
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

// Mock remotes.Fetcher interface
type mockFetcher struct {
	indexBuffers []*bytes.Buffer
}

func (f *mockFetcher) Fetch(ctx context.Context, desc ocischemav1.Descriptor) (io.ReadCloser, error) {
	rc := ioutil.NopCloser(f.indexBuffers[0])
	f.indexBuffers = f.indexBuffers[1:]
	return rc, nil
}
