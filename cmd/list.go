package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0funct0ry/maelsink/internal/cliclient"
)

var (
	listFlags   *apiClientFlags
	listQ       string
	listFrom    string
	listTo      string
	listSubject string
	listLimit   int
	listOffset  int
	listSince   string
	listUntil   string
	listSort    string
)

// listCmd lists messages via the REST API (SPEC.md §7.3).
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List messages via the REST API",
	Long: `Thin REST API client: lists stored messages in table or JSON format.

--format also accepts a Go template, docker-CLI-style, executed once per
message: maelsink list --format '{{.ID}}: {{.From}} -> {{.Subject}}'`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listFlags = addAPIClientFlags(listCmd)
	addFormatFlag(listCmd, listFlags)
	listCmd.Flags().StringVar(&listQ, "q", "", "full-text search query")
	listCmd.Flags().StringVar(&listFrom, "from", "", "filter by from address substring")
	listCmd.Flags().StringVar(&listTo, "to", "", "filter by to address substring")
	listCmd.Flags().StringVar(&listSubject, "subject", "", "filter by subject substring")
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "max messages to return (0 = server default)")
	listCmd.Flags().IntVar(&listOffset, "offset", 0, "pagination offset")
	listCmd.Flags().StringVar(&listSince, "since", "", "only messages received at/after this RFC3339 timestamp")
	listCmd.Flags().StringVar(&listUntil, "until", "", "only messages received at/before this RFC3339 timestamp")
	listCmd.Flags().StringVar(&listSort, "sort", "", "sort order: received_at_desc|received_at_asc")
}

func runList(cmd *cobra.Command, args []string) error {
	resp, err := listFlags.client().List(cmd.Context(), cliclient.ListParams{
		Query: listQ, From: listFrom, To: listTo, Subject: listSubject,
		Limit: listLimit, Offset: listOffset, Since: listSince, Until: listUntil, Sort: listSort,
	})
	if err != nil {
		return apiError(err)
	}

	switch listFlags.format {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(resp)

	case "table":
		if len(resp.Messages) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No messages.")
			return nil
		}
		cliclient.RenderTable(cmd.OutOrStdout(), resp.Messages)
		return nil

	default:
		// Any other --format value is a Go template, docker-CLI-style
		// (`docker ps --format '{{.ID}}'`), executed once per message.
		return cliclient.RenderTemplate(cmd.OutOrStdout(), resp.Messages, listFlags.format)
	}
}
