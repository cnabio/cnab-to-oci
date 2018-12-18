package remotes

import (
	"testing"

	"gotest.tools/assert"
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
