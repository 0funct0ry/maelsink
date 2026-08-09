package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0funct0ry/maelsink/internal/cliclient"
)

var (
	clearFlags *apiClientFlags
	clearYes   bool
)

// clearCmd deletes all messages via the REST API (SPEC.md §7.3).
var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete all messages via the REST API",
	Long:  `Thin REST API client: deletes every stored message. Prompts for confirmation unless --yes is given.`,
	RunE:  runClear,
}

func init() {
	rootCmd.AddCommand(clearCmd)
	clearFlags = addAPIClientFlags(clearCmd)
	clearCmd.Flags().BoolVarP(&clearYes, "yes", "y", false, "skip the confirmation prompt")
}

func runClear(cmd *cobra.Command, args []string) error {
	client := clearFlags.client()

	if !clearYes {
		resp, err := client.List(cmd.Context(), cliclient.ListParams{Limit: 1})
		if err != nil {
			return apiError(err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "This will delete %d messages. Continue? [y/N] ", resp.Total)
		reader := bufio.NewReader(cmd.InOrStdin())
		line, _ := reader.ReadString('\n')
		answer := strings.ToLower(strings.TrimSpace(line))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
			return nil
		}
	}

	if err := client.Clear(cmd.Context()); err != nil {
		return apiError(err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "All messages deleted.")
	return nil
}
