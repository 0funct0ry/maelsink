package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	exportFlags  *apiClientFlags
	exportOutput string
)

// exportCmd downloads a message as a .eml file via the REST API (SPEC.md §7.3).
var exportCmd = &cobra.Command{
	Use:   "export <id>",
	Short: "Download a message as a .eml file via the REST API",
	Long:  `Thin REST API client: writes a message's raw source to -o <path>, or ./<id>.eml if omitted.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runExport,
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportFlags = addAPIClientFlags(exportCmd)
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file path (default ./<id>.eml)")
}

func runExport(cmd *cobra.Command, args []string) error {
	id := args[0]
	raw, err := exportFlags.client().ExportRaw(cmd.Context(), id)
	if err != nil {
		return apiError(err)
	}

	path := exportOutput
	if path == "" {
		path = fmt.Sprintf("./%s.eml", id)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("error: writing %s: %w", path, err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), path)
	return nil
}
