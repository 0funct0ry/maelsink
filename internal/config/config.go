// Package config implements maelsink's layered configuration model
// (SPEC.md §3): built-in defaults < config file < MAELSINK_* env vars < CLI
// flags, in increasing order of precedence.
package config

import (
	"bytes"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type SMTPAuth struct {
	Enabled  bool   `yaml:"enabled" mapstructure:"enabled"`
	Username string `yaml:"username" mapstructure:"username"`
	Password string `yaml:"password" mapstructure:"password"`
}

type SMTP struct {
	Host             string   `yaml:"host" mapstructure:"host"`
	Port             int      `yaml:"port" mapstructure:"port"`
	Domain           string   `yaml:"domain" mapstructure:"domain"`
	MaxMessageSizeMB int      `yaml:"max_message_size_mb" mapstructure:"max_message_size_mb"`
	StartTLS         bool     `yaml:"starttls" mapstructure:"starttls"`
	TLSCert          string   `yaml:"tls_cert" mapstructure:"tls_cert"`
	TLSKey           string   `yaml:"tls_key" mapstructure:"tls_key"`
	Auth             SMTPAuth `yaml:"auth" mapstructure:"auth"`
}

type Web struct {
	Enabled     bool     `yaml:"enabled" mapstructure:"enabled"`
	Host        string   `yaml:"host" mapstructure:"host"`
	Port        int      `yaml:"port" mapstructure:"port"`
	BasePath    string   `yaml:"base_path" mapstructure:"base_path"`
	CORSOrigins []string `yaml:"cors_origins" mapstructure:"cors_origins"`
	Auth        WebAuth  `yaml:"auth" mapstructure:"auth"`
}

type WebAuth struct {
	File string `yaml:"file" mapstructure:"file"`
}

type APIAuth struct {
	Enabled bool   `yaml:"enabled" mapstructure:"enabled"`
	APIKey  string `yaml:"api_key" mapstructure:"api_key"`
}

type API struct {
	Host     string  `yaml:"host" mapstructure:"host"`
	Port     int     `yaml:"port" mapstructure:"port"`
	BasePath string  `yaml:"base_path" mapstructure:"base_path"`
	Auth     APIAuth `yaml:"auth" mapstructure:"auth"`
}

type Retention struct {
	MaxMessages          int `yaml:"max_messages" mapstructure:"max_messages"`
	MaxAgeHours          int `yaml:"max_age_hours" mapstructure:"max_age_hours"`
	SweepIntervalMinutes int `yaml:"sweep_interval_minutes" mapstructure:"sweep_interval_minutes"`
}

type Attachments struct {
	StoreOnDisk bool   `yaml:"store_on_disk" mapstructure:"store_on_disk"`
	DiskPath    string `yaml:"disk_path" mapstructure:"disk_path"`
}

type Storage struct {
	Driver      string      `yaml:"driver" mapstructure:"driver"`
	Path        string      `yaml:"path" mapstructure:"path"`
	Retention   Retention   `yaml:"retention" mapstructure:"retention"`
	Attachments Attachments `yaml:"attachments" mapstructure:"attachments"`
}

type Logging struct {
	Level  string `yaml:"level" mapstructure:"level"`
	Format string `yaml:"format" mapstructure:"format"`
	File   string `yaml:"file" mapstructure:"file"`
}

type Server struct {
	ShutdownTimeoutSeconds int `yaml:"shutdown_timeout_seconds" mapstructure:"shutdown_timeout_seconds"`
}

type Shell struct {
	CommandPrefix       string `yaml:"command_prefix" mapstructure:"command_prefix"`
	Prompt              string `yaml:"prompt" mapstructure:"prompt"`
	HistoryFile         string `yaml:"history_file" mapstructure:"history_file"`
	HistorySize         int    `yaml:"history_size" mapstructure:"history_size"`
	Color               string `yaml:"color" mapstructure:"color"`
	Seed                int64  `yaml:"seed" mapstructure:"seed"`
	Editor              string `yaml:"editor" mapstructure:"editor"`
	ShEnabled           bool   `yaml:"sh_enabled" mapstructure:"sh_enabled"`
	ExitOnError         bool   `yaml:"exit_on_error" mapstructure:"exit_on_error"`
	AbbrTriggerKey      string `yaml:"abbr_trigger_key" mapstructure:"abbr_trigger_key"`
	TemplateEnabled     bool   `yaml:"template_enabled" mapstructure:"template_enabled"`
	TemplateUnsafeFuncs bool   `yaml:"template_unsafe_funcs" mapstructure:"template_unsafe_funcs"`
}

// Config is the fully-resolved maelsink configuration (SPEC.md §3.1).
type Config struct {
	SMTP    SMTP    `yaml:"smtp" mapstructure:"smtp"`
	Web     Web     `yaml:"web" mapstructure:"web"`
	API     API     `yaml:"api" mapstructure:"api"`
	Storage Storage `yaml:"storage" mapstructure:"storage"`
	Logging Logging `yaml:"logging" mapstructure:"logging"`
	Server  Server  `yaml:"server" mapstructure:"server"`
	Shell   Shell   `yaml:"shell" mapstructure:"shell"`
}

// Defaults returns maelsink's built-in defaults (SPEC.md §3.1), the lowest
// layer of precedence — always safe to run with zero config.
//
// Hosts default to 127.0.0.1 (loopback-only) rather than 0.0.0.0, so a bare
// `maelsink serve` never listens beyond localhost by accident (SPEC.md §12).
// Container deployments (SPEC.md §9.2) override these via MAELSINK_*_HOST
// env vars baked into the image, since Docker port-mapping requires the
// process inside the container to bind 0.0.0.0.
func Defaults() Config {
	return Config{
		SMTP: SMTP{
			Host:             "127.0.0.1",
			Port:             1025,
			Domain:           "maelsink.local",
			MaxMessageSizeMB: 25,
			StartTLS:         false,
		},
		Web: Web{
			Enabled:     true,
			Host:        "127.0.0.1",
			Port:        8080,
			CORSOrigins: []string{},
			Auth:        WebAuth{File: ""},
		},
		API: API{
			Host: "127.0.0.1",
			Port: 9090,
		},
		Storage: Storage{
			Driver: "sqlite",
			Path:   "./maelsink.db",
			Attachments: Attachments{
				StoreOnDisk: false,
				DiskPath:    "./attachments",
			},
			Retention: Retention{
				SweepIntervalMinutes: 5,
			},
		},
		Logging: Logging{
			Level:  "info",
			Format: "text",
		},
		Server: Server{
			ShutdownTimeoutSeconds: 15,
		},
		Shell: Shell{
			Prompt:          "maelsink{{ if not .connected }} (offline){{ end }}> ",
			HistorySize:     5000,
			Color:           "auto",
			ShEnabled:       true,
			AbbrTriggerKey:  "space",
			TemplateEnabled: true,
		},
	}
}

// FlagOverrides carries CLI-flag-sourced values, the highest-precedence
// layer. Every field is a pointer so only flags the caller actually set
// (i.e. Cobra's Changed==true) are applied — an unset flag must never
// stomp a value from the file/env layers with its zero value.
type FlagOverrides struct {
	SMTPHost                      *string
	SMTPPort                      *int
	SMTPDomain                    *string
	SMTPMaxMessageSizeMB          *int
	SMTPStartTLS                  *bool
	SMTPTLSCert                   *string
	SMTPTLSKey                    *string
	SMTPAuthEnabled               *bool
	SMTPAuthUsername              *string
	SMTPAuthPassword              *string
	WebEnabled                    *bool
	WebHost                       *string
	WebPort                       *int
	WebBasePath                   *string
	WebCORSOrigins                *[]string
	WebAuthFile                   *string
	APIHost                       *string
	APIPort                       *int
	APIBasePath                   *string
	APIAuthEnabled                *bool
	APIAuthAPIKey                 *string
	DBPath                        *string
	StorageDriver                 *string
	StorageAttachmentsStoreOnDisk *bool
	StorageAttachmentsDiskPath    *string
	LogLevel                      *string
	LogFormat                     *string
	LogFile                       *string
	RetentionMaxMessages          *int
	RetentionMaxAgeHours          *int
	RetentionSweepIntervalMinutes *int
	ServerShutdownTimeoutSeconds  *int
	ShellCommandPrefix            *string
	ShellPrompt                   *string
	ShellHistoryFile              *string
	ShellHistorySize              *int
	ShellColor                    *string
	ShellSeed                     *int64
	ShellEditor                   *string
	ShellShEnabled                *bool
	ShellExitOnError              *bool
	ShellAbbrTriggerKey           *string
	ShellTemplateEnabled          *bool
	ShellTemplateUnsafeFuncs      *bool
}

// Options controls a single Load call.
type Options struct {
	// ConfigFile, if set, is read explicitly (--config/-c). If unset, viper
	// looks for ./maelsink.yaml and silently continues if it's absent —
	// missing config file is not an error, per "always safe to run with
	// zero config".
	ConfigFile string
	Flags      FlagOverrides
}

// Load resolves the effective Config by applying, in increasing order of
// precedence: built-in defaults, the YAML config file, MAELSINK_* env vars,
// then CLI flags.
func Load(opts Options) (Config, error) {
	cfg, _, err := load(opts)
	return cfg, err
}

// LoadWithProvenance behaves like Load but additionally resolves per-key
// Provenance (SPEC.md §8.1's Settings screen source/origin columns) via
// ResolveProvenance, using flagSet to detect which flags were explicitly
// set (Changed==true).
func LoadWithProvenance(opts Options, flagSet *pflag.FlagSet) (Config, Provenance, error) {
	cfg, v, err := load(opts)
	if err != nil {
		return Config{}, nil, err
	}
	return cfg, ResolveProvenance(v, flagSet, ProvenanceKeys()), nil
}

// load is Load's shared implementation, additionally returning the *viper.
// Viper instance used to resolve cfg so ResolveProvenance can inspect it
// (v.InConfig, v.ConfigFileUsed) after the fact.
func load(opts Options) (Config, *viper.Viper, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	setDefaults(v, Defaults())

	if opts.ConfigFile != "" {
		v.SetConfigFile(opts.ConfigFile)
		if err := v.ReadInConfig(); err != nil {
			return Config{}, nil, fmt.Errorf("config: reading %q: %w", opts.ConfigFile, err)
		}
	} else {
		v.SetConfigName("maelsink")
		v.AddConfigPath(".")
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return Config{}, nil, fmt.Errorf("config: reading maelsink.yaml: %w", err)
			}
		}
	}

	v.SetEnvPrefix("MAELSINK")
	v.SetEnvKeyReplacer(newEnvReplacer())
	v.AutomaticEnv()
	bindEnv(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, nil, fmt.Errorf("config: unmarshal: %w", err)
	}

	applyFlagOverrides(&cfg, opts.Flags)

	return cfg, v, nil
}

// Validate performs basic sanity checks on a resolved Config.
func (c Config) Validate() error {
	if c.SMTP.Port <= 0 || c.SMTP.Port > 65535 {
		return fmt.Errorf("smtp.port: invalid port %d", c.SMTP.Port)
	}
	if c.Web.Port <= 0 || c.Web.Port > 65535 {
		return fmt.Errorf("web.port: invalid port %d", c.Web.Port)
	}
	if c.API.Port <= 0 || c.API.Port > 65535 {
		return fmt.Errorf("api.port: invalid port %d", c.API.Port)
	}
	switch c.Logging.Format {
	case "text", "json":
	default:
		return fmt.Errorf("logging.format: must be text or json, got %q", c.Logging.Format)
	}
	switch c.Logging.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("logging.level: must be debug|info|warn|error, got %q", c.Logging.Level)
	}
	if c.Storage.Path == "" {
		return fmt.Errorf("storage.path: must not be empty")
	}
	if c.SMTP.MaxMessageSizeMB <= 0 {
		return fmt.Errorf("smtp.max_message_size_mb: must be > 0, got %d", c.SMTP.MaxMessageSizeMB)
	}
	if c.SMTP.StartTLS {
		if c.SMTP.TLSCert == "" || c.SMTP.TLSKey == "" {
			return fmt.Errorf("smtp.starttls: tls_cert and tls_key are both required when starttls is enabled")
		}
	}
	if c.SMTP.Auth.Enabled {
		if c.SMTP.Auth.Username == "" || c.SMTP.Auth.Password == "" {
			return fmt.Errorf("smtp.auth: username and password are both required when auth.enabled is true")
		}
	}
	switch c.Shell.Color {
	case "auto", "always", "never":
	default:
		return fmt.Errorf("shell.color: must be auto|always|never, got %q", c.Shell.Color)
	}
	switch c.Shell.AbbrTriggerKey {
	case "space", "tab", "enter", "none":
	default:
		return fmt.Errorf("shell.abbr_trigger_key: must be space|tab|enter|none, got %q", c.Shell.AbbrTriggerKey)
	}
	return nil
}

// YAML marshals the config back to YAML, matching the maelsink.yaml layout.
func (c Config) YAML() ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(c); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteFile writes the config as YAML to path, refusing to overwrite an
// existing file unless force is true.
func WriteFile(path string, c Config, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config: %s already exists (use --force to overwrite)", path)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	data, err := c.YAML()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadFile reads and parses a YAML config file directly, without layering
// (used by `config validate`).
func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	cfg := Defaults()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
