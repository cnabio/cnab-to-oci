package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/docker/cnab-to-oci/remotes"
	"github.com/docker/distribution/reference"
	"github.com/spf13/cobra"
)

type pullOptions struct {
	output             string
	targetRef          string
	insecureRegistries []string
	raw                bool
}

func pullCmd() *cobra.Command {
	var opts pullOptions
	cmd := &cobra.Command{
		Use:   "pull <ref> [options]",
		Short: "Pulls an image reference",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return runPull(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.output, "output", "o", "pulled.json", "Output file")
	cmd.Flags().StringSliceVar(&opts.insecureRegistries, "insecure-registries", nil, "Use plain HTTP for those registries")
	cmd.Flags().BoolVarP(&opts.raw, "raw", "", false, "If true, pull and write the raw manifest from the registry")
	return cmd
}

func runPull(opts pullOptions) error {
	ref, err := reference.ParseNormalizedNamed(opts.targetRef)
	if err != nil {
		return err
	}

	if opts.raw {
		index, err := remotes.GetIndex(context.Background(), ref, createResolver(opts.insecureRegistries).Resolver)
		if err != nil {
			return err
		}

		bytes, err := json.MarshalIndent(index, "", "\t")
		if err != nil {
			return err
		}
		return ioutil.WriteFile(opts.output, bytes, 0644)
	}

	b, err := remotes.Pull(context.Background(), ref, createResolver(opts.insecureRegistries).Resolver)
	if err != nil {
		return err
	}
	bytes, err := json.MarshalIndent(b, "", "\t")
	if err != nil {
		return err
	}
	if opts.output == "-" {
		fmt.Fprintln(os.Stdout, string(bytes))
		return nil
	}
	return ioutil.WriteFile(opts.output, bytes, 0644)
}
