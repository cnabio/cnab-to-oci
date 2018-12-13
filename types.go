package oci

import (
	"github.com/deislabs/duffle/pkg/bundle"
)

// BundleConfig discribes a cnab bundle runtime config
type BundleConfig struct {
	SchemaVersion string                                `json:"schema_version" mapstructure:"schema_version"`
	Actions       map[string]bundle.Action              `json:"actions,omitempty" mapstructure:"actions,omitempty"`
	Parameters    map[string]bundle.ParameterDefinition `json:"parameters" mapstructure:"parameters"`
	Credentials   map[string]bundle.Location            `json:"credentials" mapstructure:"credentials"`
}

// CreateBundleConfig creates a bundle config from a CNAB
func CreateBundleConfig(b *bundle.Bundle) *BundleConfig {
	return &BundleConfig{
		SchemaVersion: CNABVersion,
		Actions:       b.Actions,
		Parameters:    b.Parameters,
		Credentials:   b.Credentials,
	}
}
