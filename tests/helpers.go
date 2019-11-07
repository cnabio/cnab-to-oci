package tests

import (
	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-go/bundle/definition"
	"github.com/docker/cnab-to-oci/relocation"
	"github.com/docker/distribution/manifest/schema2"
	ocischema "github.com/opencontainers/image-spec/specs-go"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// MakeTestBundle creates a simple bundle for tests
func MakeTestBundle() *bundle.Bundle {
	return &bundle.Bundle{
		SchemaVersion: "v1.0.0",
		Actions: map[string]bundle.Action{
			"action-1": {
				Modifies: true,
			},
		},
		Credentials: map[string]bundle.Credential{
			"cred-1": {
				Location: bundle.Location{
					EnvironmentVariable: "env-var",
					Path:                "/some/path",
				},
			},
		},
		Description: "description",
		Images: map[string]bundle.Image{
			"image-1": {
				BaseImage: bundle.BaseImage{
					Image:     "my.registry/namespace/image-1",
					ImageType: "oci",
					MediaType: "application/vnd.oci.image.manifest.v1+json",
					Size:      507,
					Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
				},
			},
			"another-image": {
				BaseImage: bundle.BaseImage{
					Image:     "my.registry/namespace/another-image",
					ImageType: "oci",
					MediaType: "application/vnd.oci.image.manifest.v1+json",
					Size:      507,
					Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0342",
				},
			},
		},
		InvocationImages: []bundle.InvocationImage{
			{
				BaseImage: bundle.BaseImage{
					Image:     "my.registry/namespace/my-app-invoc",
					ImageType: "docker",
					MediaType: "application/vnd.docker.distribution.manifest.v2+json",
					Size:      506,
					Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0343",
				},
			},
		},
		Keywords: []string{"keyword1", "keyword2"},
		Maintainers: []bundle.Maintainer{
			{
				Email: "docker@docker.com",
				Name:  "docker",
				URL:   "docker.com",
			},
		},
		Name: "my-app",
		Definitions: map[string]*definition.Schema{
			"param1Type": {
				Enum:    []interface{}{"value1", true, float64(1)},
				Type:    []interface{}{"string", "boolean", "number"},
				Default: "hello",
			},
			"output1Type": {
				Type: "string",
			},
		},
		Parameters: map[string]bundle.Parameter{
			"param1": {
				Definition: "param1Type",
				Destination: &bundle.Location{
					EnvironmentVariable: "env_var",
					Path:                "/some/path",
				},
			},
		},
		Outputs: map[string]bundle.Output{
			"output1": {
				Definition:  "output1Type",
				Description: "magic",
				ApplyTo:     []string{"install"},
				Path:        "/cnab/app/outputs/magic",
			},
		},
		Custom: map[string]interface{}{
			"my-key": "my-value",
		},
		Version: "0.1.0",
	}
}

// MakeTestOCIIndex creates a dummy OCI index for tests
func MakeTestOCIIndex() *ocischemav1.Index {
	return &ocischemav1.Index{
		Versioned: ocischema.Versioned{
			SchemaVersion: 2,
		},
		Annotations: map[string]string{
			"io.cnab.runtime_version":         "v1.0.0",
			ocischemav1.AnnotationTitle:       "my-app",
			ocischemav1.AnnotationVersion:     "0.1.0",
			ocischemav1.AnnotationDescription: "description",
			ocischemav1.AnnotationAuthors:     `[{"name":"docker","email":"docker@docker.com","url":"docker.com"}]`,
			"io.cnab.keywords":                `["keyword1","keyword2"]`,
			"org.opencontainers.artifactType": "application/vnd.cnab.manifest.v1",
		},
		Manifests: []ocischemav1.Descriptor{
			{
				Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
				MediaType: schema2.MediaTypeManifest,
				Size:      315,
				Annotations: map[string]string{
					"io.cnab.manifest.type": "config",
				},
			},
			{
				Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0343",
				MediaType: "application/vnd.docker.distribution.manifest.v2+json",
				Size:      506,
				Annotations: map[string]string{
					"io.cnab.manifest.type": "invocation",
				},
			},
			{
				Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0342",
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Size:      507,
				Annotations: map[string]string{
					"io.cnab.manifest.type":  "component",
					"io.cnab.component.name": "another-image",
				},
			},
			{
				Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Size:      507,
				Annotations: map[string]string{
					"io.cnab.manifest.type":  "component",
					"io.cnab.component.name": "image-1",
				},
			},
		},
	}
}

// MakeRelocationMap generates a fake relocation map
func MakeRelocationMap() relocation.ImageRelocationMap {
	return relocation.ImageRelocationMap{
		"my.registry/namespace/image-1":       "my.registry/namespace/my-app@sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
		"my.registry/namespace/another-image": "my.registry/namespace/my-app@sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0342",
		"my.registry/namespace/my-app-invoc":  "my.registry/namespace/my-app@sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0343",
	}
}
