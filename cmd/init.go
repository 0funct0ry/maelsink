package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0funct0ry/maelsink/internal/config"
)

var initForce bool

// initCmd writes a default maelsink.yaml to the current directory.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Write a default maelsink.yaml to the current directory",
	Long:  `Writes maelsink's built-in default configuration to ./maelsink.yaml, refusing to overwrite an existing file unless --force is given.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		const path = "maelsink.yaml"
		if err := config.WriteFile(path, config.Defaults(), initForce); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", path)
		return nil
	},
}

func init() {
	configCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "overwrite an existing maelsink.yaml")
}
