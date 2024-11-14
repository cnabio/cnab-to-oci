package remotes

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/remotes"
	"github.com/docker/docker/api/types/image"
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
	if descriptor.Size == -1 {
		return "", descriptor, fmt.Errorf("empty descriptor")
	}
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
	returnErrorValues []error
}

func newMockPusher(ret []error) *mockPusher {
	return &mockPusher{
		pushedDescriptors: []ocischemav1.Descriptor{},
		buffers:           []*bytes.Buffer{},
		returnErrorValues: ret,
	}
}

func (p *mockPusher) Push(ctx context.Context, d ocischemav1.Descriptor) (content.Writer, error) {
	p.pushedDescriptors = append(p.pushedDescriptors, d)
	buf := &bytes.Buffer{}
	p.buffers = append(p.buffers, buf)
	var err error
	if p.returnErrorValues != nil {
		err = p.returnErrorValues[0]
		p.returnErrorValues = p.returnErrorValues[1:]
	}
	return &mockWriter{
		WriteCloser: nopWriteCloser{Buffer: buf},
	}, err
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
	rc := io.NopCloser(f.indexBuffers[0])
	f.indexBuffers = f.indexBuffers[1:]
	return rc, nil
}

type mockReadCloser struct {
}

func (rc mockReadCloser) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func (rc mockReadCloser) Close() error {
	return nil
}

type mockImageClient struct {
	pushedImages int
	taggedImages map[string]string
}

func newMockImageClient() *mockImageClient {
	return &mockImageClient{taggedImages: map[string]string{}}
}

func (c *mockImageClient) ImagePush(ctx context.Context, ref string, options image.PushOptions) (io.ReadCloser, error) {
	c.pushedImages++
	return mockReadCloser{}, nil
}
func (c *mockImageClient) ImageTag(ctx context.Context, image, ref string) error {
	c.taggedImages[image] = ref
	return nil
}
