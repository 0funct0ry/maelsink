package cmd

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/config"
	"github.com/0funct0ry/maelsink/internal/shell"
	"github.com/0funct0ry/maelsink/internal/shell/builtin"
)

var (
	flagCommandPrefix, flagPrompt, flagHistoryFile, flagEditor, flagAbbrTriggerKey, flagColor string
	flagHistorySize                                                                           int
	flagSeed                                                                                  int64
	flagShEnabled, flagExitOnError, flagTemplateEnabled, flagTemplateUnsafeFuncs              bool
	flagNoTemplate                                                                            bool
	flagExecs                                                                                 []string
	flagScript                                                                                string
	shellSMTPHost                                                                             string
	shellSMTPPort                                                                             int
	shellAuthUser, shellAuthPass                                                              string
	shellSMTPTLS                                                                              string
	shellTLSInsecureSkipVerify                                                                bool

	// shellAPI, shellAPIKey, shellFormat mirror apiClientFlags' fields, but
	// are registered directly on shellCmd rather than via
	// addAPIClientFlags/addFormatFlag: those helpers register --api/
	// --api-key/--format via plain StringVar (no shorthand), while
	// SPEC.md §3.3.1's flag table requires -A/-k/-f shorthands specifically
	// for shellCmd. shellCmd defines its own vars with the same flag names
	// and defaults so it can carry those shorthands.
	shellAPI, shellAPIKey, shellFormat string
)

// shellCmd starts the interactive/scriptable maelsink shell (SPEC.md §7.5):
// a pure client of the existing REST API and SMTP port, offering an
// interactive REPL, -e/--execute, and --script non-interactive modes.
var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Start an interactive maelsink shell",
	Long: `Starts maelsink's interactive shell: a readline-style REPL with alias/
abbreviation/template expansion, a builtin command table (list/show/delete/
clear/export/send/stats/health/...), and non-interactive scripting modes
(-e/--execute, --script). It is a pure client of the /api/v1 REST surface
and SMTP port — it never talks to storage directly.`,
	RunE: runShell,
}

func init() {
	rootCmd.AddCommand(shellCmd)
	addConfigFlag(shellCmd)

	d := config.Defaults()

	shellCmd.Flags().StringVarP(&flagCommandPrefix, "command-prefix", "x", d.Shell.CommandPrefix, `builtin prefix: "", ".", ":" or "/"`)
	shellCmd.Flags().StringVarP(&flagPrompt, "prompt", "R", d.Shell.Prompt, "prompt template")
	shellCmd.Flags().StringVarP(&flagHistoryFile, "history-file", "Y", d.Shell.HistoryFile, `"" = platform default`)
	shellCmd.Flags().IntVarP(&flagHistorySize, "history-size", "z", d.Shell.HistorySize, "max history lines kept")
	shellCmd.Flags().StringVarP(&flagColor, "color", "L", d.Shell.Color, "auto|always|never")
	shellCmd.Flags().Int64VarP(&flagSeed, "seed", "S", d.Shell.Seed, "template PRNG seed (0 = random per session)")
	shellCmd.Flags().StringVarP(&flagEditor, "editor", "E", d.Shell.Editor, `"" = $VISUAL, $EDITOR, vi/notepad`)
	shellCmd.Flags().BoolVarP(&flagShEnabled, "sh-enabled", "X", d.Shell.ShEnabled, "allow the sh builtin")
	shellCmd.Flags().BoolVarP(&flagExitOnError, "exit-on-error", "Q", d.Shell.ExitOnError, "abort a script on first failure")
	shellCmd.Flags().StringVarP(&flagAbbrTriggerKey, "abbr-trigger-key", "G", d.Shell.AbbrTriggerKey, "space|tab|enter|none")
	shellCmd.Flags().BoolVarP(&flagTemplateEnabled, "template-enabled", "t", d.Shell.TemplateEnabled, "enable {{ }} template expansion")
	shellCmd.Flags().BoolVarP(&flagNoTemplate, "no-template", "N", false, "shorthand for --template-enabled=false")
	shellCmd.Flags().BoolVarP(&flagTemplateUnsafeFuncs, "template-unsafe-funcs", "Z", d.Shell.TemplateUnsafeFuncs, "re-enable env/expandenv/getHostByName")
	shellCmd.Flags().StringArrayVarP(&flagExecs, "execute", "e", nil, "run a command and exit (repeatable)")
	shellCmd.Flags().StringVarP(&flagScript, "script", "s", "", "run a file of commands and exit")

	shellCmd.Flags().StringVarP(&shellAPI, "api", "A", fmt.Sprintf("http://%s:%d", d.API.Host, d.API.Port), "maelsink REST API base URL")
	shellCmd.Flags().StringVarP(&shellFormat, "format", "f", "table", "session default output format: table|json|yaml")
	shellCmd.Flags().StringVarP(&shellAPIKey, "api-key", "k", "", "REST API bearer key (if api.auth.enabled)")

	shellCmd.Flags().StringVarP(&shellSMTPHost, "smtp-host", "H", d.SMTP.Host, "default SMTP target for the send builtin")
	shellCmd.Flags().IntVarP(&shellSMTPPort, "smtp-port", "p", d.SMTP.Port, "default SMTP target for the send builtin")
	shellCmd.Flags().StringVarP(&shellAuthUser, "auth-user", "U", "", "SMTP AUTH username")
	shellCmd.Flags().StringVarP(&shellAuthPass, "auth-pass", "W", "", "SMTP AUTH password")
	shellCmd.Flags().StringVar(&shellSMTPTLS, "smtp-tls", "none", "default transport security for the send builtins: none|starttls|implicit")
	shellCmd.Flags().BoolVar(&shellTLSInsecureSkipVerify, "smtp-tls-insecure-skip-verify", false, "accept a self-signed/dev SMTP TLS certificate without verification (local/CI use only)")
}

// resolveShellConfig loads the layered config (defaults < file < env < flags)
// and returns just the Shell section, mirroring serve.go's resolveConfig
// pattern: only flags actually Changed on cmd override the lower layers.
func resolveShellConfig(cmd *cobra.Command) (config.Shell, error) {
	f := cmd.Flags()
	var overrides config.FlagOverrides

	if f.Changed("command-prefix") {
		overrides.ShellCommandPrefix = &flagCommandPrefix
	}
	if f.Changed("prompt") {
		overrides.ShellPrompt = &flagPrompt
	}
	if f.Changed("history-file") {
		overrides.ShellHistoryFile = &flagHistoryFile
	}
	if f.Changed("history-size") {
		overrides.ShellHistorySize = &flagHistorySize
	}
	if f.Changed("color") {
		overrides.ShellColor = &flagColor
	}
	if f.Changed("seed") {
		overrides.ShellSeed = &flagSeed
	}
	if f.Changed("editor") {
		overrides.ShellEditor = &flagEditor
	}
	if f.Changed("sh-enabled") {
		overrides.ShellShEnabled = &flagShEnabled
	}
	if f.Changed("exit-on-error") {
		overrides.ShellExitOnError = &flagExitOnError
	}
	if f.Changed("abbr-trigger-key") {
		overrides.ShellAbbrTriggerKey = &flagAbbrTriggerKey
	}
	templateEnabled := flagTemplateEnabled && !flagNoTemplate
	if f.Changed("template-enabled") || f.Changed("no-template") {
		overrides.ShellTemplateEnabled = &templateEnabled
	}
	if f.Changed("template-unsafe-funcs") {
		overrides.ShellTemplateUnsafeFuncs = &flagTemplateUnsafeFuncs
	}

	cfg, err := config.Load(config.Options{ConfigFile: cfgFile, Flags: overrides})
	if err != nil {
		return config.Shell{}, err
	}
	return cfg.Shell, nil
}

func runShell(cmd *cobra.Command, args []string) error {
	shellCfg, err := resolveShellConfig(cmd)
	if err != nil {
		return err
	}

	client := cliclient.NewClient(shellAPI, shellAPIKey)

	addr := net.JoinHostPort(shellSMTPHost, strconv.Itoa(shellSMTPPort))
	var auth *cliclient.Auth
	if shellAuthUser != "" {
		auth = &cliclient.Auth{Username: shellAuthUser, Password: shellAuthPass}
	}

	tlsMode, err := cliclient.ParseTLSMode(shellSMTPTLS)
	if err != nil {
		return fmt.Errorf("shell: %w", err)
	}
	tlsOpts := cliclient.TLSOptions{Mode: tlsMode, InsecureSkipVerify: shellTLSInsecureSkipVerify}

	interactive := len(flagExecs) == 0 && flagScript == "" &&
		isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())

	registry := shell.NewRegistry(builtin.All()...)

	exitCode, runErr := shell.Run(cmd.Context(), shell.Options{
		Cfg:         shellCfg,
		Client:      client,
		SMTPAddr:    addr,
		SMTPAuth:    auth,
		SMTPTLS:     tlsOpts,
		Registry:    registry,
		Execs:       flagExecs,
		ScriptPath:  flagScript,
		Stdin:       cmd.InOrStdin(),
		Stdout:      cmd.OutOrStdout(),
		Stderr:      cmd.ErrOrStderr(),
		Interactive: interactive,
		NewEditor:   shell.DefaultNewEditor(shellCfg),
	})

	if runErr == nil && exitCode == 0 {
		return nil
	}
	if exitCode == 1 && runErr != nil {
		return runErr
	}
	os.Exit(exitCode)
	return nil // unreachable
}
