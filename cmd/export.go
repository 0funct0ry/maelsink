package cmd

import (
	"github.com/spf13/cobra"
)

// exportCmd downloads a message as a .eml file via the REST API (SPEC.md §7.3).
var exportCmd = &cobra.Command{
	Use:   "export <id>",
	Short: "Download a message as a .eml file via the REST API",
	Long:  `Thin REST API client: writes a message's raw source to -o <path>, or ./<id>.eml if omitted.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented(cmd, "M4.0")
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
}
