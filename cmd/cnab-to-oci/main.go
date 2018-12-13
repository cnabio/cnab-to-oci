package main

import (
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use: "cnab-to-oci <subcommand> [options]",
	}
	cmd.AddCommand(fixupCmd(), pushCmd(), pullCmd())
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
