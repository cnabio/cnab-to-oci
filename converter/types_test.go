package converter

import (
	"testing"

	"github.com/deislabs/cnab-go/bundle"
	"gotest.tools/assert"
)

func TestPrepareForPush(t *testing.T) {
	b := &bundle.Bundle{}
	prepared, err := PrepareForPush(b)
	assert.NilError(t, err)

	// First try with OCI format and specific CNAB media type. Fallback should be set.
	assert.Equal(t, prepared.ManifestDescriptor.MediaType, "application/vnd.oci.image.manifest.v1+json")
	assert.Equal(t, prepared.ConfigBlobDescriptor.MediaType, "application/vnd.cnab.config.v1+json")
	assert.Check(t, prepared.Fallback != nil)
	// Try the first fallback, which set the media type to image config and still using OCI format
	fallback := prepared.Fallback
	assert.Equal(t, fallback.ManifestDescriptor.MediaType, "application/vnd.oci.image.manifest.v1+json")
	assert.Equal(t, fallback.ConfigBlobDescriptor.MediaType, "application/vnd.oci.image.config.v1+json")
	assert.Check(t, fallback.Fallback != nil)
	// Last fallback uses Docker format
	lastFallback := fallback.Fallback
	assert.Equal(t, lastFallback.ManifestDescriptor.MediaType, "application/vnd.docker.distribution.manifest.v2+json")
	assert.Equal(t, lastFallback.ConfigBlobDescriptor.MediaType, "application/vnd.docker.container.image.v1+json")
}
