package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/0funct0ry/maelsink/internal/cliclient"
)

var getFlags *apiClientFlags

// getCmd shows full message detail via the REST API (SPEC.md §7.3).
var getCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show full message detail via the REST API",
	Long:  `Thin REST API client: fetches and prints one message by id.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runGet,
}

func init() {
	rootCmd.AddCommand(getCmd)
	getFlags = addAPIClientFlags(getCmd)
	addFormatFlag(getCmd, getFlags)
}

func runGet(cmd *cobra.Command, args []string) error {
	msg, err := getFlags.client().Get(cmd.Context(), args[0])
	if err != nil {
		return apiError(err)
	}

	switch getFlags.format {
	case "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(msg)

	case "table":
		cliclient.RenderDetail(cmd.OutOrStdout(), msg)
		return nil

	default:
		// Any other --format value is a Go template, docker-CLI-style,
		// executed once against the message detail.
		return cliclient.RenderDetailTemplate(cmd.OutOrStdout(), msg, getFlags.format)
	}
}
