package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cnab-to-oci/remotes"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/registry"
	"github.com/spf13/cobra"
)

type fixupOptions struct {
	input     string
	output    string
	targetRef string
}

func fixupCmd() *cobra.Command {
	var opts fixupOptions
	cmd := &cobra.Command{
		Use:  "fixup <bundle file> [options]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.input = args[0]
			return runFixup(opts)
		},
	}
	cmd.Flags().StringVarP(&opts.output, "output", "o", "fixed-bundle.json", "specify the output file")
	cmd.Flags().StringVarP(&opts.targetRef, "target", "t", "", "reference where the bundle will be pushed")
	return cmd
}

func createResolver() docker.ResolverBlobMounter {
	cfg := config.LoadDefaultConfigFile(os.Stderr)
	return docker.NewResolver(docker.ResolverOptions{
		Authorizer: docker.NewAuthorizer(nil, func(hostName string) (string, string, error) {
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
		}),
	})
}

func runFixup(opts fixupOptions) error {
	var b bundle.Bundle
	bundleJSON, err := ioutil.ReadFile(opts.input)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bundleJSON, &b); err != nil {
		return err
	}
	resolver := createResolver()
	ref, err := reference.ParseNormalizedNamed(opts.targetRef)
	if err != nil {
		return err
	}
	if err := remotes.FixupBundle(context.Background(), &b, ref, resolver); err != nil {
		return err
	}
	bundleJSON, err = json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(opts.output, bundleJSON, 0644)
}
