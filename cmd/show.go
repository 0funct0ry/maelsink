package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// showCmd prints the fully resolved effective config (defaults < file < env
// < flags) as YAML, for debugging precedence issues.
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Print the fully resolved effective configuration",
	Long:  `Loads maelsink.yaml (or --config), layers MAELSINK_* env vars and any flags given here, and prints the result as YAML.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _, err := resolveConfig(cmd)
		if err != nil {
			return err
		}
		data, err := cfg.YAML()
		if err != nil {
			return err
		}
		fmt.Fprint(cmd.OutOrStdout(), string(data))
		return nil
	},
}

func init() {
	configCmd.AddCommand(showCmd)
	// Accept the same non-logging flags as `serve` so `config show` can
	// demonstrate/debug the full defaults < file < env < flags precedence
	// chain. --log-level/--log-format/--log-file are deliberately excluded:
	// `config show` never constructs a logger, so that CLI-flag layer is
	// reserved for `serve`/bare `maelsink` per addLogFlags' doc comment.
	addServeFlags(showCmd)
	addConfigFlag(showCmd)
}
