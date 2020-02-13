package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-to-oci/remotes"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

type pushOptions struct {
	input               string
	targetRef           string
	insecureRegistries  []string
	allowFallbacks      bool
	invocationPlatforms []string
	componentPlatforms  []string
	autoUpdateBundle    bool
	pushImages          bool
}

func pushCmd() *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:   "push <bundle file> [options]",
		Short: "Fixes and pushes the bundle to an registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.input = args[0]
			if opts.targetRef == "" {
				return errors.New("--target flag must be set with a namespace ")
			}
			return runPush(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.targetRef, "target", "t", "", "reference where the bundle will be pushed")
	cmd.Flags().StringSliceVar(&opts.insecureRegistries, "insecure-registries", nil, "Use plain HTTP for those registries")
	cmd.Flags().BoolVar(&opts.allowFallbacks, "allow-fallbacks", true, "Enable automatic compatibility fallbacks for registries without support for custom media type, or OCI manifests")
	cmd.Flags().StringSliceVar(&opts.invocationPlatforms, "invocation-platforms", nil, "Platforms to push (for multi-arch invocation images)")
	cmd.Flags().StringSliceVar(&opts.componentPlatforms, "component-platforms", nil, "Platforms to push (for multi-arch component images)")
	cmd.Flags().BoolVar(&opts.autoUpdateBundle, "auto-update-bundle", false, "Updates the bundle image properties with the one resolved on the registry")
	cmd.Flags().BoolVar(&opts.pushImages, "push-images", true, "Allow to push missing images in the registry that are available in the local docker daemon image store")

	return cmd
}

func runPush(opts pushOptions) error {
	var b bundle.Bundle
	bundleJSON, err := ioutil.ReadFile(opts.input)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bundleJSON, &b); err != nil {
		return err
	}
	resolver := createResolver(opts.insecureRegistries)
	ref, err := reference.ParseNormalizedNamed(opts.targetRef)
	if err != nil {
		return err
	}

	fixupOptions := []remotes.FixupOption{
		remotes.WithEventCallback(displayEvent),
		remotes.WithInvocationImagePlatforms(opts.invocationPlatforms),
		remotes.WithComponentImagePlatforms(opts.componentPlatforms),
	}
	if opts.autoUpdateBundle {
		fixupOptions = append(fixupOptions, remotes.WithAutoBundleUpdate())
	}
	if opts.pushImages {
		cli, err := client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			return err
		}
		fixupOptions = append(fixupOptions, remotes.WithPushImages(cli, os.Stdout))
	}
	relocationMap, err := remotes.FixupBundle(context.Background(), &b, ref, resolver, fixupOptions...)
	if err != nil {
		return err
	}
	d, err := remotes.Push(context.Background(), &b, relocationMap, ref, resolver, opts.allowFallbacks)
	if err != nil {
		return err
	}
	fmt.Printf("Pushed successfully, with digest %q\n", d.Digest)
	return nil
}
