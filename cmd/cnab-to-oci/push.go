package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/cnab-to-oci/remotes"
	"github.com/docker/distribution/reference"
	"github.com/spf13/cobra"
)

type pushOptions struct {
	input     string
	targetRef string
}

func pushCmd() *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:   "push <bundle file> [options]",
		Short: "Fixes and pushes the bundle to an registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.input = args[0]
			return runPush(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.targetRef, "target", "t", "", "reference where the bundle will be pushed")
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
	resolver := createResolver()
	ref, err := reference.ParseNormalizedNamed(opts.targetRef)
	if err != nil {
		return err
	}
	if err := remotes.FixupBundle(context.Background(), &b, ref, resolver); err != nil {
		return err
	}
	d, err := remotes.Push(context.Background(), &b, ref, resolver)
	if err != nil {
		return err
	}
	fmt.Printf("Pushed successfully, with digest %q\n", d.Digest)
	return nil
}
