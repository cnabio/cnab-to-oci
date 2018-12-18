package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func versionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Shows the version of cnab-to-oci",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("TODO(ulyssessouza) Implement versioning")
			return nil
		},
	}
	return cmd
}
