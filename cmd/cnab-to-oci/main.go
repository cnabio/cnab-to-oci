package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:          "cnab-to-oci <subcommand> [options]",
		SilenceUsage: true,
	}
	cmd.AddCommand(fixupCmd(), pushCmd(), pullCmd())
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
