package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/cnabio/cnab-to-oci/remotes"
	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
	"github.com/distribution/reference"
	"github.com/spf13/cobra"
)

type pullOptions struct {
	bundle             string
	relocationMap      string
	targetRef          string
	insecureRegistries []string
}

func pullCmd() *cobra.Command {
	var opts pullOptions
	cmd := &cobra.Command{
		Use:   "pull <ref> [options]",
		Short: "Pulls an image reference",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return runPull(opts)
		},
	}

	cmd.Flags().StringVar(&opts.bundle, "bundle", "pulled.json", "bundle output file (- to print on standard output)")
	cmd.Flags().StringVar(&opts.relocationMap, "relocation-map", "relocation-map.json", "relocation map output file (- to print on standard output)")
	cmd.Flags().StringSliceVar(&opts.insecureRegistries, "insecure-registries", nil, "Use plain HTTP for those registries")
	return cmd
}

func runPull(opts pullOptions) error {
	ref, err := reference.ParseNormalizedNamed(opts.targetRef)
	if err != nil {
		return err
	}

	b, relocationMap, _, err := remotes.Pull(context.Background(), ref, createResolver(opts.insecureRegistries))
	if err != nil {
		return err
	}
	if err := writeOutput(opts.bundle, b); err != nil {
		return err
	}
	return writeOutput(opts.relocationMap, relocationMap)
}

func writeOutput(file string, data interface{}) error {
	plainJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}
	bytes, err := jsoncanonicalizer.Transform(plainJSON)
	if err != nil {
		return err
	}
	if file == "-" {
		fmt.Fprintln(os.Stdout, string(bytes))
		return nil
	}
	return os.WriteFile(file, bytes, 0644)
}
