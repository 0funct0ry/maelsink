package cmd

import (
	"fmt"
	"os"
	"strings"

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
	rootCmd.SetArgs(normalizeBareDBFlag(os.Args[1:]))
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// normalizeBareDBFlag rewrites a trailing/unaccompanied "--db"/"-d" (no
// argument at all — the last token, or immediately followed by another
// flag) to "--db=", before cobra/pflag ever parses args.
//
// This exists instead of pflag's NoOptDefVal mechanism because
// NoOptDefVal's substitution isn't limited to the "no argument given" case:
// once set, pflag *always* uses NoOptDefVal for "--db value"/"-d value"
// (space-separated) too, since it can't tell a following positional-looking
// token from that value's own intended argument — the same ambiguity every
// optional-value flag has (a bare "--db" is genuinely indistinguishable
// from "--db /path/to/file" without a rule for which token is which). This
// rewrite keeps that documented space-separated form ("--db ./x.db")
// working exactly as before, and only special-cases the truly-bare form.
//
// A real path can't itself start with "-" without --db=-foo — an edge case
// shared with essentially every CLI flag parser, not specific to this rule.
func normalizeBareDBFlag(args []string) []string {
	out := make([]string, 0, len(args))
	for i, a := range args {
		if (a == "--db" || a == "-d") && (i+1 >= len(args) || strings.HasPrefix(args[i+1], "-")) {
			out = append(out, "--db=")
			continue
		}
		out = append(out, a)
	}
	return out
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
