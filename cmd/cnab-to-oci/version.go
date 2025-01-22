package main

import (
	"fmt"

	"github.com/cnabio/cnab-to-oci/internal"
	"github.com/spf13/cobra"
)

func versionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Shows the version of cnab-to-oci",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println(internal.FullVersion())
			return nil
		},
	}
	return cmd
}
