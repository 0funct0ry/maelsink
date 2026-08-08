package cmd

import (
	"github.com/spf13/cobra"
)

// configCmd is the parent for config subcommands: init, show, validate.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage maelsink configuration",
	Long:  `Subcommands for generating, inspecting, and validating maelsink.yaml.`,
}

func init() {
	rootCmd.AddCommand(configCmd)
}
