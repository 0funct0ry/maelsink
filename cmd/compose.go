package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/0funct0ry/maelsink/internal/compose"
	"github.com/0funct0ry/maelsink/internal/config"
	"github.com/0funct0ry/maelsink/internal/logging"
)

var (
	composeListen                string
	composeAPIAddr               string
	composeAPIUser               string
	composeAPIPass               string
	composeAPIInsecureSkipVerify bool
	composeAPICACert             string
	composeSMTPAddr              string
	composeSMTPUser              string
	composeSMTPPass              string
	composeOpen                  bool
)

// composeCmd starts `maelsink compose` (SPEC.md §7.7): a standalone local web
// server/SPA giving a browser-based, point-and-click front end to a running
// target maelsink instance's SMTP and REST API surfaces. It is a pure client
// of that target, like shellCmd — it starts no database of its own and never
// imports internal/store/internal/smtp/internal/api/internal/webui.
var composeCmd = &cobra.Command{
	Use:   "compose",
	Short: "Start the maelsink compose browser-based playground",
	Long: `Starts maelsink compose: a small local web server serving a standalone
single-page app that gives every capability of "maelsink shell" a visual,
point-and-click front end. It is a pure client of a target maelsink
instance's REST API and SMTP port — it starts no database and never talks
to storage directly. Useful for trying maelsink for the first time, or for
any target (e.g. a headless Docker deployment) with no local terminal
attached.`,
	RunE: runCompose,
}

func init() {
	rootCmd.AddCommand(composeCmd)
	addComposeFlags(composeCmd)
}

func addComposeFlags(cmd *cobra.Command) {
	d := config.Defaults()
	cmd.Flags().StringVarP(&composeListen, "listen", "L", d.Compose.Listen, "compose server listen address")
	cmd.Flags().StringVarP(&composeAPIAddr, "api-addr", "A", d.Compose.APIAddr, "target maelsink REST API base URL")
	cmd.Flags().StringVarP(&composeAPIUser, "api-user", "u", d.Compose.APIUser, "basic-auth username for the target API (if fronted by auth)")
	cmd.Flags().StringVarP(&composeAPIPass, "api-pass", "P", d.Compose.APIPass, "basic-auth password for the target API")
	cmd.Flags().BoolVarP(&composeAPIInsecureSkipVerify, "api-insecure-skip-verify", "k", d.Compose.APIInsecureSkipVerify, "skip TLS verification when calling the target API (local/CI use only)")
	cmd.Flags().StringVarP(&composeAPICACert, "api-ca-cert", "C", d.Compose.APICACert, "path to a CA cert to trust for the target API's TLS")
	cmd.Flags().StringVarP(&composeSMTPAddr, "smtp-addr", "S", d.Compose.SMTPAddr, "target maelsink SMTP address (host:port)")
	cmd.Flags().StringVarP(&composeSMTPUser, "smtp-user", "U", d.Compose.SMTPUser, "target SMTP AUTH username")
	cmd.Flags().StringVarP(&composeSMTPPass, "smtp-pass", "W", d.Compose.SMTPPass, "target SMTP AUTH password")
	cmd.Flags().BoolVarP(&composeOpen, "open", "o", d.Compose.Open, "automatically open the compose UI in a browser on startup")
}

// resolveComposeConfig loads the layered config (defaults < file < env <
// flags), mirroring resolveShellConfig's pattern in cmd/shell.go.
func resolveComposeConfig(cmd *cobra.Command) (config.Config, error) {
	f := cmd.Flags()
	var overrides config.FlagOverrides

	if f.Changed("listen") {
		overrides.ComposeListen = &composeListen
	}
	if f.Changed("api-addr") {
		overrides.ComposeAPIAddr = &composeAPIAddr
	}
	if f.Changed("api-user") {
		overrides.ComposeAPIUser = &composeAPIUser
	}
	if f.Changed("api-pass") {
		overrides.ComposeAPIPass = &composeAPIPass
	}
	if f.Changed("api-insecure-skip-verify") {
		overrides.ComposeAPIInsecureSkipVerify = &composeAPIInsecureSkipVerify
	}
	if f.Changed("api-ca-cert") {
		overrides.ComposeAPICACert = &composeAPICACert
	}
	if f.Changed("smtp-addr") {
		overrides.ComposeSMTPAddr = &composeSMTPAddr
	}
	if f.Changed("smtp-user") {
		overrides.ComposeSMTPUser = &composeSMTPUser
	}
	if f.Changed("smtp-pass") {
		overrides.ComposeSMTPPass = &composeSMTPPass
	}
	if f.Changed("open") {
		overrides.ComposeOpen = &composeOpen
	}
	if logLevel != "" {
		overrides.LogLevel = &logLevel
	}
	if logFormat != "" {
		overrides.LogFormat = &logFormat
	}
	if logFile != "" {
		overrides.LogFile = &logFile
	}

	return config.Load(config.Options{ConfigFile: cfgFile, Flags: overrides})
}

func runCompose(cmd *cobra.Command, args []string) error {
	cfg, err := resolveComposeConfig(cmd)
	if err != nil {
		return err
	}
	composeCfg := cfg.Compose

	logger, err := logging.New(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.File)
	if err != nil {
		return fmt.Errorf("logging: %w", err)
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client, err := compose.NewTargetClient(compose.TargetConfig{
		APIAddr:               composeCfg.APIAddr,
		APIUser:               composeCfg.APIUser,
		APIPass:               composeCfg.APIPass,
		APIInsecureSkipVerify: composeCfg.APIInsecureSkipVerify,
		APICACert:             composeCfg.APICACert,
		SMTPAddr:              composeCfg.SMTPAddr,
		SMTPUser:              composeCfg.SMTPUser,
		SMTPPass:              composeCfg.SMTPPass,
	})
	if err != nil {
		return fmt.Errorf("compose: building target client: %w", err)
	}

	router := compose.New(client, logger, compose.Config{})

	srv := &http.Server{Addr: composeCfg.Listen, Handler: router}

	fmt.Fprintf(cmd.OutOrStdout(), "  Compose  -> http://%s/\n", effectiveOpenAddr(composeCfg.Listen))
	fmt.Fprintf(cmd.OutOrStdout(), "  Target   -> %s (SMTP %s)\n", composeCfg.APIAddr, composeCfg.SMTPAddr)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	if composeCfg.Open {
		go openBrowser("http://" + effectiveOpenAddr(composeCfg.Listen))
	}

	select {
	case <-ctx.Done():
		return srv.Shutdown(context.Background())
	case err := <-errCh:
		return err
	}
}

// effectiveOpenAddr turns a ":8090"-style listen address into a browser-
// friendly "localhost:8090" host:port.
func effectiveOpenAddr(listen string) string {
	if strings.HasPrefix(listen, ":") {
		return "localhost" + listen
	}
	return listen
}

// openBrowser shells out to the platform's "open a URL" command. Errors are
// intentionally ignored — failing to auto-open a browser is never fatal to
// compose starting up.
func openBrowser(url string) {
	var args []string
	switch runtime.GOOS {
	case "darwin":
		args = []string{"open", url}
	case "windows":
		args = []string{"rundll32", "url.dll,FileProtocolHandler", url}
	default:
		args = []string{"xdg-open", url}
	}
	_ = exec.Command(args[0], args[1:]...).Start()
}
