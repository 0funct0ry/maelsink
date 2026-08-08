package cmd

import (
	"github.com/spf13/cobra"
)

var jsonOutput bool

// versionCmd prints build provenance per SPEC.md §7.3a.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Prints maelsink's version, commit, and Go runtime version.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return printVersion(cmd, jsonOutput)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolVar(&jsonOutput, "json", false, "print version info as JSON")
}
