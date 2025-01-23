package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-to-oci/remotes"
	containerdRemotes "github.com/containerd/containerd/remotes"
	"github.com/distribution/reference"
	"github.com/docker/cli/cli/config"
	"github.com/spf13/cobra"
)

type fixupOptions struct {
	input              string
	bundle             string
	relocationMap      string
	targetRef          string
	insecureRegistries []string
	autoUpdateBundle   bool
}

func fixupCmd() *cobra.Command {
	var opts fixupOptions
	cmd := &cobra.Command{
		Use:   "fixup <bundle file> [options]",
		Short: "Fixes the digest of an image",
		Long:  "The fixup command resolves all the digest references from a registry and patches the bundle.json with them.",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			opts.input = args[0]
			return runFixup(opts)
		},
	}
	cmd.Flags().StringVar(&opts.bundle, "bundle", "fixed-bundle.json", "fixed bundle output file (- to print on standard output)")
	cmd.Flags().StringVar(&opts.relocationMap, "relocation-map", "relocation-map.json", "relocation map output file (- to print on standard output)")
	cmd.Flags().StringVarP(&opts.targetRef, "target", "t", "", "reference where the bundle will be pushed")
	cmd.Flags().StringSliceVar(&opts.insecureRegistries, "insecure-registries", nil, "Use plain HTTP for those registries")
	cmd.Flags().BoolVar(&opts.autoUpdateBundle, "auto-update-bundle", false, "Updates the bundle image properties with the one resolved on the registry")
	return cmd
}

func runFixup(opts fixupOptions) error {
	bundleJSON, err := os.ReadFile(opts.input)
	if err != nil {
		return err
	}

	b, err := bundle.Unmarshal(bundleJSON)
	if err != nil {
		return err
	}

	ref, err := reference.ParseNormalizedNamed(opts.targetRef)
	if err != nil {
		return err
	}

	fixupOptions := []remotes.FixupOption{
		remotes.WithEventCallback(displayEvent),
	}
	if opts.autoUpdateBundle {
		fixupOptions = append(fixupOptions, remotes.WithAutoBundleUpdate())
	}
	relocationMap, err := remotes.FixupBundle(context.Background(), b, ref, createResolver(opts.insecureRegistries), fixupOptions...)
	if err != nil {
		return err
	}
	if err := writeOutput(opts.bundle, b); err != nil {
		return err
	}
	return writeOutput(opts.relocationMap, relocationMap)
}

func displayEvent(ev remotes.FixupEvent) {
	switch ev.EventType {
	case remotes.FixupEventTypeCopyImageStart:
		fmt.Fprintf(os.Stderr, "Starting to copy image %s...\n", ev.SourceImage)
	case remotes.FixupEventTypeCopyImageEnd:
		if ev.Error != nil {
			fmt.Fprintf(os.Stderr, "Failed to copy image %s: %s\n", ev.SourceImage, ev.Error)
		} else {
			fmt.Fprintf(os.Stderr, "Completed image %s copy\n", ev.SourceImage)
		}
	}
}

func createResolver(insecureRegistries []string) containerdRemotes.Resolver {
	return remotes.CreateResolver(config.LoadDefaultConfigFile(os.Stderr), insecureRegistries...)
}
