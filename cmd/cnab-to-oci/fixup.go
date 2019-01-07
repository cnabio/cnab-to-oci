package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cnab-to-oci/remotes"
	"github.com/docker/distribution/reference"
	"github.com/spf13/cobra"
)

type fixupOptions struct {
	input     string
	output    string
	targetRef string
	insecure  bool
}

func fixupCmd() *cobra.Command {
	var opts fixupOptions
	cmd := &cobra.Command{
		Use:   "fixup <bundle file> [options]",
		Short: "Fixes the digest of an image",
		Long:  "The fixup command resolves all the digest references from a registry and patches the bundle.json with them.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.input = args[0]
			return runFixup(opts)
		},
	}
	cmd.Flags().StringVarP(&opts.output, "output", "o", "fixed-bundle.json", "specify the output file")
	cmd.Flags().StringVarP(&opts.targetRef, "target", "t", "", "reference where the bundle will be pushed")
	cmd.Flags().BoolVar(&opts.insecure, "insecure", false, "Use insecure registry, without SSL")
	return cmd
}

func createResolver(plainHTTP bool) docker.ResolverBlobMounter {
	cfg := config.LoadDefaultConfigFile(os.Stderr)
	return remotes.CreateResolver(cfg, plainHTTP)
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
	resolver := createResolver(opts.insecure)
	ref, err := reference.ParseNormalizedNamed(opts.targetRef)
	if err != nil {
		return err
	}
	if err := remotes.FixupBundle(context.Background(), &b, ref, resolver); err != nil {
		return err
	}
	bundleJSON, err = json.MarshalIndent(b, "", "\t")
	if err != nil {
		return err
	}
	if opts.output == "-" {
		fmt.Fprintln(os.Stdout, string(bundleJSON))
		return nil
	}
	return ioutil.WriteFile(opts.output, bundleJSON, 0644)
}
