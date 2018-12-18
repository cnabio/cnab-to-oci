package oci

import (
	"testing"

	"github.com/docker/cnab-to-oci/test"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"gotest.tools/assert"
)

func makeTestBundleConfig() *BundleConfig {
	return CreateBundleConfig(test.MakeTestBundle())
}

func TestConvertFromFixedUpBundleToOCI(t *testing.T) {
	bundleConfigDescriptor := ocischemav1.Descriptor{
		Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
		MediaType: schema2.MediaTypeManifest,
		Size:      250,
	}
	targetRef := "my.registry/namespace/my-app:0.1.0"
	src := test.MakeTestBundle()

	expected := test.MakeTestOCIIndex()

	// Convert from bundle to OCI index
	named, err := reference.ParseNormalizedNamed(targetRef)
	assert.NilError(t, err)
	actual, err := ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor)
	assert.NilError(t, err)
	assert.DeepEqual(t, expected, actual)

	// Nil maintainers does not add annotation
	src = test.MakeTestBundle()
	src.Maintainers = nil
	actual, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor)
	assert.NilError(t, err)
	_, hasMaintainers := actual.Annotations[ocischemav1.AnnotationAuthors]
	assert.Assert(t, !hasMaintainers)

	// Nil keywords does not add annotation
	src = test.MakeTestBundle()
	src.Keywords = nil
	actual, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor)
	assert.NilError(t, err)
	_, hasKeywords := actual.Annotations[CNABKeywordsAnnotation]
	assert.Assert(t, !hasKeywords)

	// Multiple invocation images is not supported
	src = test.MakeTestBundle()
	src.InvocationImages = append(src.InvocationImages, src.InvocationImages[0])
	_, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor)
	assert.ErrorContains(t, err, "only one invocation image supported")

	// Invalid media type
	src = test.MakeTestBundle()
	src.InvocationImages[0].MediaType = "some-invalid-mediatype"
	_, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor)
	assert.ErrorContains(t, err, "unsupported media type \"some-invalid-mediatype\" for image \"my.registry/namespace/my-app@sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341\"")

	// All images must be in the same repository
	src = test.MakeTestBundle()
	src.InvocationImages[0].BaseImage.Image = "my.registry/namespace/other-repo@sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341"
	_, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor)
	assert.ErrorContains(t, err, "invalid invocation image: image \"my.registry/namespace/other-repo@sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341\" is not in the same repository as \"my.registry/namespace/my-app:0.1.0\"")

	// Image reference must be digested
	src = test.MakeTestBundle()
	src.InvocationImages[0].BaseImage.Image = "my.registry/namespace/my-app:not-digested"
	_, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor)
	assert.ErrorContains(t, err, "invalid invocation image: image \"my.registry/namespace/my-app:not-digested\" is not a digested reference")

	// Invalid reference
	src = test.MakeTestBundle()
	src.InvocationImages[0].BaseImage.Image = "Some/iNvalid/Ref"
	_, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor)
	assert.ErrorContains(t, err, "invalid invocation image: image \"Some/iNvalid/Ref\" is not a valid image reference: invalid reference format: repository name must be lowercase")

	//  mediatype ociindex
	src = test.MakeTestBundle()
	src.InvocationImages[0].MediaType = ocischemav1.MediaTypeImageIndex
	_, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor)
	assert.NilError(t, err)

	// mediatype docker manifestlist
	src = test.MakeTestBundle()
	src.InvocationImages[0].MediaType = "application/vnd.docker.distribution.manifest.list.v2+json"
	_, err = ConvertBundleToOCIIndex(src, named, bundleConfigDescriptor)
	assert.NilError(t, err)
}

func TestGetConfigDescriptor(t *testing.T) {
	ix := &ocischemav1.Index{
		Manifests: []ocischemav1.Descriptor{
			ocischemav1.Descriptor{
				Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
				MediaType: schema2.MediaTypeManifest,
				Size:      250,
				Annotations: map[string]string{
					CNABDescriptorTypeAnnotation: CNABDescriptorTypeConfig,
				},
			},
			ocischemav1.Descriptor{
				Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
				MediaType: "application/vnd.docker.distribution.manifest.v2+json",
				Size:      250,
				Annotations: map[string]string{
					"io.cnab.type": "invocation",
				},
			},
			ocischemav1.Descriptor{
				Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Size:      250,
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
		Size:      250,
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

func TestConvertFromOCIToBundle(t *testing.T) {
	targetRef := "my.registry/namespace/my-app:0.1.0"
	named, err := reference.ParseNormalizedNamed(targetRef)
	assert.NilError(t, err)
	ix := test.MakeTestOCIIndex()
	config := makeTestBundleConfig()
	expected := test.MakeTestBundle()

	result, err := ConvertOCIIndexToBundle(ix, config, named)
	assert.NilError(t, err)
	assert.DeepEqual(t, expected, result)

	// Without title annotation
	delete(ix.Annotations, ocischemav1.AnnotationTitle)
	_, err = ConvertOCIIndexToBundle(ix, config, named)
	assert.ErrorContains(t, err, "manifest is missing title annotation")

	// Without version annotation
	ix = test.MakeTestOCIIndex()
	delete(ix.Annotations, ocischemav1.AnnotationVersion)
	_, err = ConvertOCIIndexToBundle(ix, config, named)
	assert.ErrorContains(t, err, "manifest is missing version annotation")

	// Invalid authors annotation
	ix = test.MakeTestOCIIndex()
	ix.Annotations[ocischemav1.AnnotationAuthors] = "Some garbage"
	_, err = ConvertOCIIndexToBundle(ix, config, named)
	assert.ErrorContains(t, err, "unable to parse maintainers")

	// Invalid keywords annotation
	ix = test.MakeTestOCIIndex()
	ix.Annotations[CNABKeywordsAnnotation] = "Some garbage"
	_, err = ConvertOCIIndexToBundle(ix, config, named)
	assert.ErrorContains(t, err, "unable to parse keywords")

	// bad media type
	ix = test.MakeTestOCIIndex()
	ix.Manifests[1].MediaType = "Some garbage"
	_, err = ConvertOCIIndexToBundle(ix, config, named)
	assert.ErrorContains(t, err, "unsupported manifest descriptor")

	// no cnab type (invocation/component)
	ix = test.MakeTestOCIIndex()
	delete(ix.Manifests[1].Annotations, CNABDescriptorTypeAnnotation)
	_, err = ConvertOCIIndexToBundle(ix, config, named)
	assert.ErrorContains(t, err, "has no CNAB descriptor type annotation \"io.cnab.type\"")

	// bad cnab type
	ix = test.MakeTestOCIIndex()
	ix.Manifests[1].Annotations[CNABDescriptorTypeAnnotation] = "Some garbage"
	_, err = ConvertOCIIndexToBundle(ix, config, named)
	assert.ErrorContains(t, err, "invalid CNAB descriptor type \"Some garbage\" in descriptor")

	// component name missing
	ix = test.MakeTestOCIIndex()
	delete(ix.Manifests[2].Annotations, CNABDescriptorComponentNameAnnotation)
	_, err = ConvertOCIIndexToBundle(ix, config, named)
	assert.ErrorContains(t, err, "component name missing in descriptor")
}
