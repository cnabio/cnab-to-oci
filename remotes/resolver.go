package remotes

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/registry"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// multiRegistryResolver is an OCI registry resolver that accepts a list of
// insecure registries. It will skip TLS validation for registries that are secured with TLS
// use plain http for unsecured registries and any registry that is exposed on a loopback ip address.
type multiRegistryResolver struct {
	resolver            remotes.Resolver
	plainHTTPRegistries map[string]struct{}
	skipTLSRegistries   map[string]struct{}
	authorizer          docker.Authorizer
	skipTLSClient       *http.Client
	skipTLSAuthorizer   docker.Authorizer
}

func (r *multiRegistryResolver) Resolve(ctx context.Context, ref string) (name string, desc ocispec.Descriptor, err error) {
	name, desc, err = r.resolver.Resolve(ctx, ref)

	// Add some extra context to the poor error message
	// which is returned when you forget to specify that the registry
	// uses an insecure TLS certificate
	// Example: pulling from host localhost:55027 failed with status code [manifests sha256:464c8a63f292a07fb0ea2bf2cf636dafe38bf74d0536879fb9ec4611f2168067]: 400 Bad Request
	if err != nil && strings.Contains(err.Error(), "400 Bad Request") {
		ref, otherErr := reference.ParseNormalizedNamed(ref)
		if otherErr != nil {
			return
		}
		repoInfo, otherErr := registry.ParseRepositoryInfo(ref)
		if otherErr != nil {
			return
		}

		// Check if the registry is not flagged with skipTLS, which is one common explanation for this error
		if _, skipTLS := r.skipTLSRegistries[repoInfo.Index.Name]; !skipTLS {
			err = fmt.Errorf("possible attempt to access an insecure registry without skipping TLS verification detected: %w", err)
		}
	}

	return
}

func (r *multiRegistryResolver) Fetcher(ctx context.Context, ref string) (remotes.Fetcher, error) {
	return r.resolver.Fetcher(ctx, ref)
}

func (r *multiRegistryResolver) Pusher(ctx context.Context, ref string) (remotes.Pusher, error) {
	return r.resolver.Pusher(ctx, ref)
}

// CreateResolver creates a docker registry resolver, using the local docker CLI credentials
func CreateResolver(cfg *configfile.ConfigFile, insecureRegistries ...string) remotes.Resolver {
	authCreds := docker.WithAuthCreds(func(hostName string) (string, string, error) {
		if hostName == registry.DefaultV2Registry.Host {
			hostName = registry.IndexServer
		}
		a, err := cfg.GetAuthConfig(hostName)
		if err != nil {
			return "", "", err
		}
		if a.IdentityToken != "" {
			return "", a.IdentityToken, nil
		}
		return a.Username, a.Password, nil
	})

	clientSkipTLS := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	result := &multiRegistryResolver{
		authorizer:          docker.NewDockerAuthorizer(authCreds),
		skipTLSClient:       clientSkipTLS,
		skipTLSAuthorizer:   docker.NewDockerAuthorizer(authCreds, docker.WithAuthClient(clientSkipTLS)),
		plainHTTPRegistries: make(map[string]struct{}),
		skipTLSRegistries:   make(map[string]struct{}),
	}

	// Determine ahead of time how each registry is insecure
	// 1. It uses TLS but has a bad cert
	// 2. It doesn't use TLS
	for _, r := range insecureRegistries {
		pingURL := fmt.Sprintf("https://%s/v2/", r)
		resp, err := clientSkipTLS.Get(pingURL)
		if err == nil {
			resp.Body.Close()
			result.skipTLSRegistries[r] = struct{}{}
		} else {
			result.plainHTTPRegistries[r] = struct{}{}
		}
	}

	result.resolver = docker.NewResolver(docker.ResolverOptions{
		Hosts: result.configureHosts(),
	})

	return result
}

func (r *multiRegistryResolver) configureHosts() docker.RegistryHosts {
	return func(host string) ([]docker.RegistryHost, error) {
		config := docker.RegistryHost{
			Client:       http.DefaultClient,
			Authorizer:   r.authorizer,
			Host:         host,
			Scheme:       "https",
			Path:         "/v2",
			Capabilities: docker.HostCapabilityPull | docker.HostCapabilityResolve | docker.HostCapabilityPush,
		}

		if _, skipTLS := r.skipTLSRegistries[host]; skipTLS {
			config.Client = r.skipTLSClient
			config.Authorizer = r.skipTLSAuthorizer
		} else if _, plainHTTP := r.plainHTTPRegistries[host]; plainHTTP {
			config.Scheme = "http"
		} else {
			// Default to plain http for localhost
			match, err := docker.MatchLocalhost(host)
			if err != nil {
				return nil, err
			}
			if match {
				config.Scheme = "http"
			}
		}

		// If this is not set, then we aren't prompted to authenticate to Docker Hub,
		// which causes the returned content type to be text/html instead of the
		// specialized content types for images and manifests
		if host == "docker.io" {
			config.Host = "registry-1.docker.io"
		}

		return []docker.RegistryHost{config}, nil
	}
}
