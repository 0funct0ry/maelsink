package cmd

import (
	"github.com/spf13/cobra"
)

// deleteCmd deletes one message via the REST API (SPEC.md §7.3).
var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete one message via the REST API",
	Long:  `Thin REST API client: deletes a single message by id.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented(cmd, "M4.0")
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
