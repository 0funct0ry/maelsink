package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0funct0ry/maelsink/internal/config"
)

// validateCmd validates a config file without starting the server. It uses
// the persistent --config/-c flag defined on the root command.
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a maelsink.yaml file",
	Long:  `Loads and validates a config file (--config, defaults to ./maelsink.yaml), reporting any errors, without starting the server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path := cfgFile
		if path == "" {
			path = "maelsink.yaml"
		}

		cfg, err := config.LoadFile(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s: OK\n", path)
		return nil
	},
}

func init() {
	configCmd.AddCommand(validateCmd)
}
