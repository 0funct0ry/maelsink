package cmd

import (
	"github.com/spf13/cobra"
)

// listCmd lists messages via the REST API (SPEC.md §7.3).
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List messages via the REST API",
	Long:  `Thin REST API client: lists stored messages in table or JSON format.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented(cmd, "M4.0")
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
