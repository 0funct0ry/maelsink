package cmd

import (
	"github.com/spf13/cobra"
)

// getCmd shows full message detail via the REST API (SPEC.md §7.3).
var getCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show full message detail via the REST API",
	Long:  `Thin REST API client: fetches and prints one message by id.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return notImplemented(cmd, "M4.0")
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
