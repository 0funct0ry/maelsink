package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/0funct0ry/maelsink/internal/version"
)

var (
	cfgFile     string
	logLevel    string
	logFormat   string
	logFile     string
	showVersion bool
)

// rootCmd represents the base command when called without any subcommands.
// Running bare `maelsink [flags]` is an alias for `maelsink serve [flags]`
// (SPEC.md §7): RunE delegates to the same runServe function serveCmd uses,
// so there is exactly one implementation to keep in sync.
var rootCmd = &cobra.Command{
	Use:   "maelsink",
	Short: "A single-binary fake SMTP server for local dev and CI",
	Long: `maelsink accepts any mail over SMTP, stores it, and exposes it via a
web UI and REST API. It never relays mail anywhere — it's a sink, not an MTA.

Running "maelsink" with no subcommand is equivalent to "maelsink serve".`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			return printVersion(cmd, false)
		}
		return runServe(cmd, args)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "path to maelsink.yaml")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "", "log level (debug|info|warn|error)")
	rootCmd.PersistentFlags().StringVarP(&logFormat, "log-format", "F", "", "log format (text|json)")
	rootCmd.PersistentFlags().StringVarP(&logFile, "log-file", "j", "", "log file path (empty = stdout only)")

	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "print version information and exit")

	// serve's flags are also accepted on the bare root command, since bare
	// `maelsink` is an alias for `maelsink serve`.
	addServeFlags(rootCmd)
}

// notImplemented is the stub RunE for commands not yet built (SPEC.md's
// milestone plan) so exit code is non-zero rather than silently succeeding.
func notImplemented(cmd *cobra.Command, milestone string) error {
	fmt.Fprintf(cmd.ErrOrStderr(), "%s is not implemented yet (arrives in %s)\n", cmd.Name(), milestone)
	return fmt.Errorf("%s: not implemented", cmd.Name())
}

func printVersion(cmd *cobra.Command, asJSON bool) error {
	info := version.Get()
	if asJSON {
		fmt.Fprintf(cmd.OutOrStdout(), "{\"version\":%q,\"commit\":%q,\"build_date\":%q,\"go\":%q}\n",
			info.Version, info.Commit, info.BuildDate, info.Go)
		return nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "maelsink version %s (commit %s, built %s)\n", info.Version, info.Commit, info.BuildDate)
	return nil
}
