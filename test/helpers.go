package test

import (
	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/distribution/manifest/schema2"
	ocischema "github.com/opencontainers/image-spec/specs-go"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// MakeTestBundle creates a simple bundle for tests
func MakeTestBundle() *bundle.Bundle {
	return &bundle.Bundle{
		Actions: map[string]bundle.Action{
			"action-1": bundle.Action{
				Modifies: true,
			},
		},
		Credentials: map[string]bundle.Location{
			"cred-1": bundle.Location{
				EnvironmentVariable: "env-var",
				Path:                "/some/path",
			},
		},
		Description: "description",
		Images: map[string]bundle.Image{
			"image-1": bundle.Image{
				Description: "nginx:2.12",
				BaseImage: bundle.BaseImage{
					Image:     "my.registry/namespace/my-app@sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
					ImageType: "oci",
					MediaType: "application/vnd.oci.image.manifest.v1+json",
					Size:      250,
				},
			},
		},
		InvocationImages: []bundle.InvocationImage{
			bundle.InvocationImage{
				BaseImage: bundle.BaseImage{
					Image:     "my.registry/namespace/my-app@sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
					ImageType: "docker",
					MediaType: "application/vnd.docker.distribution.manifest.v2+json",
					Size:      250,
				},
			},
		},
		Keywords: []string{"keyword1", "keyword2"},
		Maintainers: []bundle.Maintainer{
			bundle.Maintainer{
				Email: "docker@docker.com",
				Name:  "docker",
				URL:   "docker.com",
			},
		},
		Name: "my-app",
		Parameters: map[string]bundle.ParameterDefinition{
			"param1": bundle.ParameterDefinition{
				AllowedValues: []interface{}{"value1", true, float64(1)},
				DataType:      "type",
				DefaultValue:  "hello",
				Destination: &bundle.Location{
					EnvironmentVariable: "env_var",
					Path:                "/some/path",
				},
			},
		},
		Version: "0.1.0",
	}
}

func MakeTestOCIIndex() *ocischemav1.Index {
	return &ocischemav1.Index{
		Versioned: ocischema.Versioned{
			SchemaVersion: 1,
		},
		Annotations: map[string]string{
			"io.docker.app.format":            "cnab",
			"io.cnab.runtime_version":         "v1.0.0-WD",
			ocischemav1.AnnotationTitle:       "my-app",
			ocischemav1.AnnotationVersion:     "0.1.0",
			ocischemav1.AnnotationDescription: "description",
			ocischemav1.AnnotationAuthors:     `[{"name":"docker","email":"docker@docker.com","url":"docker.com"}]`,
			"io.cnab.keywords":                `["keyword1","keyword2"]`,
		},
		Manifests: []ocischemav1.Descriptor{
			ocischemav1.Descriptor{
				Digest:    "sha256:d59a1aa7866258751a261bae525a1842c7ff0662d4f34a355d5f36826abc0341",
				MediaType: schema2.MediaTypeManifest,
				Size:      250,
				Annotations: map[string]string{
					"io.cnab.type": "config",
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
}
