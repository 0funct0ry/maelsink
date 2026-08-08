package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0funct0ry/maelsink/internal/config"
)

var (
	flagSMTPHost             string
	flagSMTPPort             int
	flagSMTPDomain           string
	flagWebEnabled           bool
	flagHeadless             bool
	flagWebHost              string
	flagWebPort              int
	flagWebBasePath          string
	flagAPIHost              string
	flagAPIPort              int
	flagAPIBasePath          string
	flagDBPath               string
	flagRetentionMaxMessages int
	flagRetentionMaxAgeHours int
)

// serveCmd starts the SMTP + Web UI + REST API stack (SPEC.md §7.1). The
// actual server implementations land in M1.0+; this milestone wires config
// resolution and flags only.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the SMTP, Web UI, and REST API servers",
	Long: `Starts maelsink's SMTP server, embedded Web UI, and REST API concurrently,
per the resolved configuration (defaults < config file < env vars < flags).`,
	RunE: runServe,
}

// addServeFlags registers serve's flags on cmd. Called for both serveCmd
// and rootCmd so that bare `maelsink [flags]` accepts the same flags as
// `maelsink serve [flags]` (SPEC.md §7).
func addServeFlags(cmd *cobra.Command) {
	d := config.Defaults()

	cmd.Flags().StringVarP(&flagSMTPHost, "smtp-host", "H", d.SMTP.Host, "SMTP listen host")
	cmd.Flags().IntVarP(&flagSMTPPort, "smtp-port", "p", d.SMTP.Port, "SMTP listen port")
	cmd.Flags().StringVarP(&flagSMTPDomain, "smtp-domain", "m", d.SMTP.Domain, "HELO/EHLO advertised domain")
	cmd.Flags().BoolVarP(&flagWebEnabled, "web-enabled", "e", d.Web.Enabled, "enable the Web UI server")
	cmd.Flags().BoolVarP(&flagHeadless, "headless", "u", false, "shorthand for --web-enabled=false (headless mode)")
	cmd.Flags().StringVarP(&flagWebHost, "web-host", "w", d.Web.Host, "Web UI listen host")
	cmd.Flags().IntVarP(&flagWebPort, "web-port", "P", d.Web.Port, "Web UI listen port")
	cmd.Flags().StringVarP(&flagWebBasePath, "web-base-path", "b", d.Web.BasePath, "Web UI reverse-proxy base path")
	cmd.Flags().StringVarP(&flagAPIHost, "api-host", "A", d.API.Host, "REST API listen host")
	cmd.Flags().IntVarP(&flagAPIPort, "api-port", "o", d.API.Port, "REST API listen port")
	cmd.Flags().StringVarP(&flagAPIBasePath, "api-base-path", "B", d.API.BasePath, "REST API reverse-proxy base path")
	cmd.Flags().StringVarP(&flagDBPath, "db", "d", d.Storage.Path, "path to the SQLite database file")
	cmd.Flags().IntVarP(&flagRetentionMaxMessages, "retention-max-messages", "M", d.Storage.Retention.MaxMessages, "max stored messages (0 = unlimited)")
	cmd.Flags().IntVarP(&flagRetentionMaxAgeHours, "retention-max-age-hours", "g", d.Storage.Retention.MaxAgeHours, "max message age in hours (0 = unlimited)")
}

func init() {
	rootCmd.AddCommand(serveCmd)
	addServeFlags(serveCmd)
}

// resolveConfig loads the layered config using the flags actually set on
// cmd (Changed==true), so unset flags never override the file/env layers
// with their zero value.
func resolveConfig(cmd *cobra.Command) (config.Config, error) {
	f := cmd.Flags()
	var overrides config.FlagOverrides

	if f.Changed("smtp-host") {
		overrides.SMTPHost = &flagSMTPHost
	}
	if f.Changed("smtp-port") {
		overrides.SMTPPort = &flagSMTPPort
	}
	if f.Changed("smtp-domain") {
		overrides.SMTPDomain = &flagSMTPDomain
	}
	webEnabled := flagWebEnabled && !flagHeadless
	if f.Changed("web-enabled") || f.Changed("headless") {
		overrides.WebEnabled = &webEnabled
	}
	if f.Changed("web-host") {
		overrides.WebHost = &flagWebHost
	}
	if f.Changed("web-port") {
		overrides.WebPort = &flagWebPort
	}
	if f.Changed("web-base-path") {
		overrides.WebBasePath = &flagWebBasePath
	}
	if f.Changed("api-host") {
		overrides.APIHost = &flagAPIHost
	}
	if f.Changed("api-port") {
		overrides.APIPort = &flagAPIPort
	}
	if f.Changed("api-base-path") {
		overrides.APIBasePath = &flagAPIBasePath
	}
	if f.Changed("db") {
		overrides.DBPath = &flagDBPath
	}
	if logLevel != "" {
		overrides.LogLevel = &logLevel
	}
	if logFormat != "" {
		overrides.LogFormat = &logFormat
	}
	if f.Changed("retention-max-messages") {
		overrides.RetentionMaxMessages = &flagRetentionMaxMessages
	}
	if f.Changed("retention-max-age-hours") {
		overrides.RetentionMaxAgeHours = &flagRetentionMaxAgeHours
	}

	return config.Load(config.Options{ConfigFile: cfgFile, Flags: overrides})
}

func runServe(cmd *cobra.Command, args []string) error {
	cfg, err := resolveConfig(cmd)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "serve is not implemented yet (arrives in M1.0) — config resolved successfully:")
	fmt.Fprintf(cmd.OutOrStdout(), "  SMTP     -> %s:%d\n", cfg.SMTP.Host, cfg.SMTP.Port)
	if cfg.Web.Enabled {
		fmt.Fprintf(cmd.OutOrStdout(), "  Web UI   -> http://%s:%d/\n", cfg.Web.Host, cfg.Web.Port)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  REST API -> http://%s:%d/api/v1\n", cfg.API.Host, cfg.API.Port)
	fmt.Fprintf(cmd.OutOrStdout(), "  Storage  -> %s (%s)\n", cfg.Storage.Path, cfg.Storage.Driver)
	return nil
}
