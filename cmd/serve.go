package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/0funct0ry/maelsink/internal/api"
	"github.com/0funct0ry/maelsink/internal/config"
	"github.com/0funct0ry/maelsink/internal/events"
	"github.com/0funct0ry/maelsink/internal/logging"
	"github.com/0funct0ry/maelsink/internal/retention"
	"github.com/0funct0ry/maelsink/internal/smtp"
	"github.com/0funct0ry/maelsink/internal/store"
	"github.com/0funct0ry/maelsink/internal/store/sqlite"
	"github.com/0funct0ry/maelsink/internal/webui"
	"github.com/0funct0ry/maelsink/internal/ws"
)

var (
	flagSMTPHost                      string
	flagSMTPPort                      int
	flagSMTPDomain                    string
	flagSMTPMaxMessageSizeMB          int
	flagSMTPStartTLS                  bool
	flagSMTPTLSCert                   string
	flagSMTPTLSKey                    string
	flagSMTPAuthEnabled               bool
	flagSMTPAuthUsername              string
	flagSMTPAuthPassword              string
	flagWebEnabled                    bool
	flagHeadless                      bool
	flagWebHost                       string
	flagWebPort                       int
	flagWebBasePath                   string
	flagWebCORSOrigins                []string
	flagAPIHost                       string
	flagAPIPort                       int
	flagAPIBasePath                   string
	flagAPIAuthEnabled                bool
	flagAPIAuthAPIKey                 string
	flagDBPath                        string
	flagStorageDriver                 string
	flagStorageAttachmentsStoreOnDisk bool
	flagStorageAttachmentsDiskPath    string
	flagLogFile                       string
	flagRetentionMaxMessages          int
	flagRetentionMaxAgeHours          int
	flagRetentionSweepIntervalMinutes int
	flagServerShutdownTimeoutSeconds  int
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
	cmd.Flags().IntVarP(&flagSMTPMaxMessageSizeMB, "smtp-max-message-size-mb", "s", d.SMTP.MaxMessageSizeMB, "max accepted message size in MB")
	cmd.Flags().BoolVarP(&flagSMTPStartTLS, "smtp-starttls", "t", d.SMTP.StartTLS, "enable optional STARTTLS")
	cmd.Flags().StringVarP(&flagSMTPTLSCert, "smtp-tls-cert", "C", d.SMTP.TLSCert, "path to the STARTTLS certificate file")
	cmd.Flags().StringVarP(&flagSMTPTLSKey, "smtp-tls-key", "K", d.SMTP.TLSKey, "path to the STARTTLS private key file")
	cmd.Flags().BoolVarP(&flagSMTPAuthEnabled, "smtp-auth-enabled", "a", d.SMTP.Auth.Enabled, "require AUTH PLAIN/LOGIN on the SMTP server")
	cmd.Flags().StringVarP(&flagSMTPAuthUsername, "smtp-auth-username", "U", d.SMTP.Auth.Username, "SMTP AUTH username")
	cmd.Flags().StringVarP(&flagSMTPAuthPassword, "smtp-auth-password", "W", d.SMTP.Auth.Password, "SMTP AUTH password")
	cmd.Flags().BoolVarP(&flagWebEnabled, "web-enabled", "e", d.Web.Enabled, "enable the Web UI server")
	cmd.Flags().BoolVarP(&flagHeadless, "headless", "u", false, "shorthand for --web-enabled=false (headless mode)")
	cmd.Flags().StringVarP(&flagWebHost, "web-host", "w", d.Web.Host, "Web UI listen host")
	cmd.Flags().IntVarP(&flagWebPort, "web-port", "P", d.Web.Port, "Web UI listen port")
	cmd.Flags().StringVarP(&flagWebBasePath, "web-base-path", "b", d.Web.BasePath, "Web UI reverse-proxy base path")
	cmd.Flags().StringSliceVarP(&flagWebCORSOrigins, "web-cors-origins", "O", d.Web.CORSOrigins, "allowed CORS origins for the Web UI server")
	cmd.Flags().StringVarP(&flagAPIHost, "api-host", "A", d.API.Host, "REST API listen host")
	cmd.Flags().IntVarP(&flagAPIPort, "api-port", "o", d.API.Port, "REST API listen port")
	cmd.Flags().StringVarP(&flagAPIBasePath, "api-base-path", "B", d.API.BasePath, "REST API reverse-proxy base path")
	cmd.Flags().BoolVarP(&flagAPIAuthEnabled, "api-auth-enabled", "y", d.API.Auth.Enabled, "require a bearer API key on /api/v1")
	cmd.Flags().StringVarP(&flagAPIAuthAPIKey, "api-auth-api-key", "k", d.API.Auth.APIKey, "REST API bearer key")
	cmd.Flags().StringVarP(&flagDBPath, "db", "d", d.Storage.Path, "path to the SQLite database file")
	cmd.Flags().StringVarP(&flagStorageDriver, "storage-driver", "r", d.Storage.Driver, "storage driver")
	cmd.Flags().BoolVarP(&flagStorageAttachmentsStoreOnDisk, "storage-attachments-store-on-disk", "n", d.Storage.Attachments.StoreOnDisk, "store attachments on disk instead of as SQLite BLOBs")
	cmd.Flags().StringVarP(&flagStorageAttachmentsDiskPath, "storage-attachments-disk-path", "x", d.Storage.Attachments.DiskPath, "directory for on-disk attachment storage")
	cmd.Flags().IntVarP(&flagRetentionMaxMessages, "retention-max-messages", "M", d.Storage.Retention.MaxMessages, "max stored messages (0 = unlimited)")
	cmd.Flags().IntVarP(&flagRetentionMaxAgeHours, "retention-max-age-hours", "g", d.Storage.Retention.MaxAgeHours, "max message age in hours (0 = unlimited)")
	cmd.Flags().IntVarP(&flagRetentionSweepIntervalMinutes, "retention-sweep-interval-minutes", "i", d.Storage.Retention.SweepIntervalMinutes, "retention sweeper interval in minutes")
	cmd.Flags().IntVarP(&flagServerShutdownTimeoutSeconds, "server-shutdown-timeout-seconds", "T", d.Server.ShutdownTimeoutSeconds, "graceful shutdown timeout in seconds")
}

func init() {
	rootCmd.AddCommand(serveCmd)
	addServeFlags(serveCmd)
}

// resolveConfig loads the layered config using the flags actually set on
// cmd (Changed==true), so unset flags never override the file/env layers
// with their zero value. It also resolves per-key Provenance (M6.1) for the
// Settings screen's GET /ui-api/v1/config endpoint.
func resolveConfig(cmd *cobra.Command) (config.Config, config.Provenance, error) {
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
	if f.Changed("smtp-max-message-size-mb") {
		overrides.SMTPMaxMessageSizeMB = &flagSMTPMaxMessageSizeMB
	}
	if f.Changed("smtp-starttls") {
		overrides.SMTPStartTLS = &flagSMTPStartTLS
	}
	if f.Changed("smtp-tls-cert") {
		overrides.SMTPTLSCert = &flagSMTPTLSCert
	}
	if f.Changed("smtp-tls-key") {
		overrides.SMTPTLSKey = &flagSMTPTLSKey
	}
	if f.Changed("smtp-auth-enabled") {
		overrides.SMTPAuthEnabled = &flagSMTPAuthEnabled
	}
	if f.Changed("smtp-auth-username") {
		overrides.SMTPAuthUsername = &flagSMTPAuthUsername
	}
	if f.Changed("smtp-auth-password") {
		overrides.SMTPAuthPassword = &flagSMTPAuthPassword
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
	if f.Changed("web-cors-origins") {
		overrides.WebCORSOrigins = &flagWebCORSOrigins
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
	if f.Changed("api-auth-enabled") {
		overrides.APIAuthEnabled = &flagAPIAuthEnabled
	}
	if f.Changed("api-auth-api-key") {
		overrides.APIAuthAPIKey = &flagAPIAuthAPIKey
	}
	if f.Changed("db") {
		overrides.DBPath = &flagDBPath
	}
	if f.Changed("storage-driver") {
		overrides.StorageDriver = &flagStorageDriver
	}
	if f.Changed("storage-attachments-store-on-disk") {
		overrides.StorageAttachmentsStoreOnDisk = &flagStorageAttachmentsStoreOnDisk
	}
	if f.Changed("storage-attachments-disk-path") {
		overrides.StorageAttachmentsDiskPath = &flagStorageAttachmentsDiskPath
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
	if f.Changed("retention-max-messages") {
		overrides.RetentionMaxMessages = &flagRetentionMaxMessages
	}
	if f.Changed("retention-max-age-hours") {
		overrides.RetentionMaxAgeHours = &flagRetentionMaxAgeHours
	}
	if f.Changed("retention-sweep-interval-minutes") {
		overrides.RetentionSweepIntervalMinutes = &flagRetentionSweepIntervalMinutes
	}
	if f.Changed("server-shutdown-timeout-seconds") {
		overrides.ServerShutdownTimeoutSeconds = &flagServerShutdownTimeoutSeconds
	}

	return config.LoadWithProvenance(config.Options{ConfigFile: cfgFile, Flags: overrides}, f)
}

func runServe(cmd *cobra.Command, args []string) error {
	cfg, provenance, err := resolveConfig(cmd)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	logger, err := logging.New(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.File)
	if err != nil {
		return fmt.Errorf("logging: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var messageStore store.MessageStore
	switch cfg.Storage.Driver {
	case "sqlite":
		db, err := sqlite.Open(ctx, cfg.Storage.Path)
		if err != nil {
			return fmt.Errorf("storage: %w", err)
		}
		defer db.Close()
		messageStore = sqlite.New(db, cfg.Storage.Attachments.StoreOnDisk, cfg.Storage.Attachments.DiskPath)
	default:
		return fmt.Errorf("storage: unknown driver %q", cfg.Storage.Driver)
	}
	// bus is the in-process event bus (SPEC.md §2.2/§5.5, M7.0): every
	// mutating path (SMTP ingestion, REST delete/clear, the retention
	// sweeper) publishes to it, and hub fans events out to connected
	// WebSocket clients.
	bus := events.NewBus()
	hub := ws.NewHub(bus, logger)

	sweeper := retention.New(messageStore, bus, retention.Config{
		MaxMessages: cfg.Storage.Retention.MaxMessages,
		MaxAgeHours: cfg.Storage.Retention.MaxAgeHours,
		Interval:    time.Duration(cfg.Storage.Retention.SweepIntervalMinutes) * time.Minute,
	}, retention.RealClock{}, logger)
	go sweeper.Run(ctx)

	smtpSrv, err := smtp.New(smtp.Config{
		Host:           cfg.SMTP.Host,
		Port:           cfg.SMTP.Port,
		Domain:         cfg.SMTP.Domain,
		MaxMessageSize: int64(cfg.SMTP.MaxMessageSizeMB) * 1024 * 1024,
		StartTLS:       cfg.SMTP.StartTLS,
		TLSCert:        cfg.SMTP.TLSCert,
		TLSKey:         cfg.SMTP.TLSKey,
		AuthEnabled:    cfg.SMTP.Auth.Enabled,
		AuthUsername:   cfg.SMTP.Auth.Username,
		AuthPassword:   cfg.SMTP.Auth.Password,
	}, messageStore, bus, logger)
	if err != nil {
		return fmt.Errorf("smtp: %w", err)
	}

	apiRouter := api.New(messageStore, bus, logger, api.Config{
		BasePath: cfg.API.BasePath,
		Auth:     api.Auth{Enabled: cfg.API.Auth.Enabled, APIKey: cfg.API.Auth.APIKey},
	})
	apiSrv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port),
		Handler: apiRouter,
	}

	var webSrv *http.Server
	if cfg.Web.Enabled {
		webuiRouter := webui.New(messageStore, bus, hub, logger, webui.Config{
			BasePath:      cfg.Web.BasePath,
			Auth:          api.Auth{Enabled: cfg.API.Auth.Enabled, APIKey: cfg.API.Auth.APIKey},
			SMTPHost:      cfg.SMTP.Host,
			SMTPPort:      cfg.SMTP.Port,
			ConfigEntries: config.Dump(cfg, provenance),
		})
		webSrv = &http.Server{
			Addr:    fmt.Sprintf("%s:%d", cfg.Web.Host, cfg.Web.Port),
			Handler: webuiRouter,
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "  SMTP     -> %s:%d\n", cfg.SMTP.Host, cfg.SMTP.Port)
	if cfg.Web.Enabled {
		fmt.Fprintf(cmd.OutOrStdout(), "  Web UI   -> http://%s:%d%s/\n", cfg.Web.Host, cfg.Web.Port, cfg.Web.BasePath)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  REST API -> http://%s:%d%s/api/v1\n", cfg.API.Host, cfg.API.Port, cfg.API.BasePath)
	fmt.Fprintf(cmd.OutOrStdout(), "  Storage  -> %s (%s)\n", cfg.Storage.Path, cfg.Storage.Driver)

	// Full graceful drain with a configurable timeout is M10.0's job; here
	// we only need the listener and its connection-handler goroutines to
	// fully stop on shutdown, per SPEC.md §2.3's goroutine-leak requirement.
	errCh := make(chan error, 1)
	go func() { errCh <- smtpSrv.ListenAndServe(ctx) }()

	apiErrCh := make(chan error, 1)
	go func() {
		if err := apiSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			apiErrCh <- err
		}
	}()

	webErrCh := make(chan error, 1)
	if webSrv != nil {
		go func() {
			if err := webSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				webErrCh <- err
			}
		}()
	}

	// Full graceful drain (draining in-flight /ws upgrades, coordinating
	// with the other listeners) is M10.0's job; here we broadcast
	// server.shutdown and close every WS client before the HTTP servers
	// tear down their listeners, since a closed http.Server may already
	// have torn down hijacked connections.
	shutdown := func() {
		hub.Shutdown(context.Background())
		_ = apiSrv.Shutdown(context.Background())
		if webSrv != nil {
			_ = webSrv.Shutdown(context.Background())
		}
	}

	select {
	case <-ctx.Done():
		shutdown()
		return smtpSrv.Close()
	case err := <-errCh:
		shutdown()
		return err
	case err := <-apiErrCh:
		shutdown()
		_ = smtpSrv.Close()
		return err
	case err := <-webErrCh:
		shutdown()
		_ = smtpSrv.Close()
		return err
	}
}
