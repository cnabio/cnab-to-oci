package converter

import (
	"testing"

	"github.com/cnabio/cnab-to-oci/tests"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"gotest.tools/assert"
)

func TestConvertFromFixedUpBundleToOCI(t *testing.T) {
	bundleConfigDescriptor := ocischemav1.Descriptor{
		Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
		MediaType: schema2.MediaTypeManifest,
		Size:      315,
	}
	targetRef := "my.registry/namespace/my-app:0.1.0"
	src := tests.MakeTestBundle()

	relocationMap := tests.MakeRelocationMap()

	expected := tests.MakeTestOCIIndex()

	// Convert from bundle to OCI index
	named, err := reference.ParseNormalizedNamed(targetRef)
	assert.NilError(t, err)
	actual, err := ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor, relocationMap)
	assert.NilError(t, err)
	assert.DeepEqual(t, expected, actual)

	// Nil maintainers does not add annotation
	src = tests.MakeTestBundle()
	src.Maintainers = nil
	actual, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor, relocationMap)
	assert.NilError(t, err)
	_, hasMaintainers := actual.Annotations[ocischemav1.AnnotationAuthors]
	assert.Assert(t, !hasMaintainers)

	// Nil keywords does not add annotation
	src = tests.MakeTestBundle()
	src.Keywords = nil
	actual, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor, relocationMap)
	assert.NilError(t, err)
	_, hasKeywords := actual.Annotations[CNABKeywordsAnnotation]
	assert.Assert(t, !hasKeywords)

	// Multiple invocation images is not supported
	src = tests.MakeTestBundle()
	src.InvocationImages = append(src.InvocationImages, src.InvocationImages[0])
	_, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor, relocationMap)
	assert.ErrorContains(t, err, "only one invocation image supported")

	// Invalid media type
	src = tests.MakeTestBundle()
	src.InvocationImages[0].MediaType = "some-invalid-mediatype"
	_, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor, relocationMap)
	assert.ErrorContains(t, err, `unsupported media type "some-invalid-mediatype" for image "my.registry/namespace/my-app@sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0343"`)

	// All images must be in the same repository
	src = tests.MakeTestBundle()
	badRelocationMap := tests.MakeRelocationMap()
	badRelocationMap["my.registry/namespace/my-app-invoc"] = "my.registry/namespace/other-repo@sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0343"
	_, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor, badRelocationMap)
	assert.ErrorContains(t, err, `invalid invocation image: image `+
		`"my.registry/namespace/other-repo@sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0343" is not in the same repository as "my.registry/namespace/my-app:0.1.0"`)

	// Image reference must be digested
	src = tests.MakeTestBundle()
	badRelocationMap = tests.MakeRelocationMap()
	badRelocationMap["my.registry/namespace/my-app-invoc"] = "my.registry/namespace/my-app:not-digested"
	_, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor, badRelocationMap)
	assert.ErrorContains(t, err, "invalid invocation image: image \"my.registry/namespace/"+
		"my-app:not-digested\" is not a digested reference")

	// Invalid reference
	src = tests.MakeTestBundle()
	badRelocationMap = tests.MakeRelocationMap()
	badRelocationMap["my.registry/namespace/my-app-invoc"] = "Some/iNvalid/Ref"
	_, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor, badRelocationMap)
	assert.ErrorContains(t, err, "invalid invocation image: "+
		"image \"Some/iNvalid/Ref\" is not a valid image reference: invalid reference format: repository name must be lowercase")

	// Invalid size
	src = tests.MakeTestBundle()
	src.InvocationImages[0].Size = 0
	_, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor, relocationMap)
	assert.ErrorContains(t, err, "size is not set")

	// mediatype ociindex
	src = tests.MakeTestBundle()
	src.InvocationImages[0].MediaType = ocischemav1.MediaTypeImageIndex
	_, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor, relocationMap)
	assert.NilError(t, err)

	// mediatype docker manifestlist
	src = tests.MakeTestBundle()
	src.InvocationImages[0].MediaType = "application/vnd.docker.distribution.manifest.list.v2+json"
	_, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor, relocationMap)
	assert.NilError(t, err)
}

func TestGetConfigDescriptor(t *testing.T) {
	ix := &ocischemav1.Index{
		Manifests: []ocischemav1.Descriptor{
			{
				Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
				MediaType: schema2.MediaTypeManifest,
				Size:      315,
				Annotations: map[string]string{
					CNABDescriptorTypeAnnotation: CNABDescriptorTypeConfig,
				},
			},
			{
				Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
				MediaType: "application/vnd.docker.distribution.manifest.v2+json",
				Size:      315,
				Annotations: map[string]string{
					"io.cnab.type": "invocation",
				},
			},
			{
				Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Size:      1385,
				Annotations: map[string]string{
					"io.cnab.type":           "component",
					"io.cnab.component_name": "image-1",
					"io.cnab.original_name":  "nginx:2.12",
				},
			},
		},
	}
	expected := ocischemav1.Descriptor{
		Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
		MediaType: schema2.MediaTypeManifest,
		Size:      315,
		Annotations: map[string]string{
			CNABDescriptorTypeAnnotation: CNABDescriptorTypeConfig,
		},
	}
	d, err := GetBundleConfigManifestDescriptor(ix)
	assert.NilError(t, err)
	assert.DeepEqual(t, expected, d)
	ix.Manifests = ix.Manifests[1:]
	_, err = GetBundleConfigManifestDescriptor(ix)
	assert.ErrorContains(t, err, "bundle config not found")
}

func TestGenerateRelocationMap(t *testing.T) {
	targetRef := "my.registry/namespace/my-app:0.1.0"
	named, err := reference.ParseNormalizedNamed(targetRef)
	assert.NilError(t, err)

	ix := tests.MakeTestOCIIndex()
	b := tests.MakeTestBundle()

	expected := tests.MakeRelocationMap()

	relocationMap, err := GenerateRelocationMap(ix, b, named)
	assert.NilError(t, err)
	assert.DeepEqual(t, relocationMap, expected)
}
