package cmd

import (
	"github.com/spf13/cobra"
)

// clearCmd deletes all messages via the REST API (SPEC.md §7.3).
var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete all messages via the REST API",
	Long:  `Thin REST API client: deletes every stored message. Prompts for confirmation unless --yes is given.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented(cmd, "M4.0")
	},
}

func init() {
	rootCmd.AddCommand(clearCmd)
}
