package remotes

import (
	"github.com/distribution/reference"
	registrytypes "github.com/moby/moby/api/types/registry"
)

const (
	// DefaultDomain is the default domain used for images on Docker Hub
	defaultDomain = "docker.io"

	// LegacyDefaultDomain is the legacy domain for Docker Hub
	legacyDefaultDomain = "index.docker.io"

	// DefaultRegistryHost is the actual host for Docker Hub registry
	defaultRegistryHost = "registry-1.docker.io"
)

// IndexInfo contains information about a registry
type IndexInfo struct {
	Name     string
	Official bool
}

// RepositoryInfo contains information about a repository
type RepositoryInfo struct {
	Index *IndexInfo
}

// ParseRepositoryInfo parses a repository reference and returns repository information
// This is a simplified replacement for registry.ParseRepositoryInfo from docker/docker
func ParseRepositoryInfo(ref reference.Named) (*RepositoryInfo, error) {
	domain := reference.Domain(ref)

	indexInfo := &IndexInfo{
		Name:     domain,
		Official: false,
	}

	// Check if this is an official Docker Hub registry
	if domain == defaultDomain || domain == legacyDefaultDomain {
		indexInfo.Official = true
		indexInfo.Name = legacyDefaultDomain // Use legacy domain for auth
	}

	return &RepositoryInfo{
		Index: indexInfo,
	}, nil
}

// GetAuthConfigHostname returns the hostname to use for authentication
func GetAuthConfigHostname(index *registrytypes.IndexInfo) string {
	if index.Official {
		return legacyDefaultDomain
	}
	return index.Name
}
