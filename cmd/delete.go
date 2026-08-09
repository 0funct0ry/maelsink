package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteFlags *apiClientFlags

// deleteCmd deletes one message via the REST API (SPEC.md §7.3).
var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete one message via the REST API",
	Long:  `Thin REST API client: deletes a single message by id.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteFlags = addAPIClientFlags(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
	if err := deleteFlags.client().Delete(cmd.Context(), args[0]); err != nil {
		return apiError(err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "deleted %s\n", args[0])
	return nil
}
