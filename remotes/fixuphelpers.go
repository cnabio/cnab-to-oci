package remotes

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"

	"github.com/containerd/containerd/remotes"
	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type sourceFetcherAdder interface {
	remotes.Fetcher
	Add(data []byte) digest.Digest
}

type sourceFetcherWithLocalData struct {
	inner     remotes.Fetcher
	localData map[digest.Digest][]byte
}

func newSourceFetcherWithLocalData(inner remotes.Fetcher) *sourceFetcherWithLocalData {
	return &sourceFetcherWithLocalData{
		inner:     inner,
		localData: make(map[digest.Digest][]byte),
	}
}

func (s *sourceFetcherWithLocalData) Add(data []byte) digest.Digest {
	d := digest.FromBytes(data)
	s.localData[d] = data
	return d
}

func (s *sourceFetcherWithLocalData) Fetch(ctx context.Context, desc ocischemav1.Descriptor) (io.ReadCloser, error) {
	if v, ok := s.localData[desc.Digest]; ok {
		return ioutil.NopCloser(bytes.NewReader(v)), nil
	}
	return s.inner.Fetch(ctx, desc)
}

type imageFixupInfo struct {
	targetRepo         reference.Named
	sourceRef          reference.Named
	resolvedDescriptor ocischemav1.Descriptor
}
