package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	var logLevel string
	cmd := &cobra.Command{
		Use:          "cnab-to-oci <subcommand> [options]",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			level, err := logrus.ParseLevel(logLevel)
			if err != nil {
				return err
			}
			logrus.SetLevel(level)
			return nil
		},
	}
	cmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", `Set the logging level ("debug"|"info"|"warn"|"error"|"fatal")`)
	cmd.AddCommand(fixupCmd(), pushCmd(), pullCmd(), versionCmd())
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
