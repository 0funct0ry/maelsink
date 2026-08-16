package config

import (
	"strings"

	"github.com/spf13/viper"
)

// setDefaults registers every default value with viper so the "defaults"
// layer is always present, even with no file/env/flags at all.
func setDefaults(v *viper.Viper, d Config) {
	v.SetDefault("smtp.host", d.SMTP.Host)
	v.SetDefault("smtp.port", d.SMTP.Port)
	v.SetDefault("smtp.domain", d.SMTP.Domain)
	v.SetDefault("smtp.max_message_size_mb", d.SMTP.MaxMessageSizeMB)
	v.SetDefault("smtp.require_starttls", d.SMTP.RequireStartTLS)
	v.SetDefault("smtp.require_tls", d.SMTP.RequireTLS)
	v.SetDefault("smtp.tls_cert", d.SMTP.TLSCert)
	v.SetDefault("smtp.tls_key", d.SMTP.TLSKey)
	v.SetDefault("smtp.auth.enabled", d.SMTP.Auth.Enabled)
	v.SetDefault("smtp.auth.username", d.SMTP.Auth.Username)
	v.SetDefault("smtp.auth.password", d.SMTP.Auth.Password)
	v.SetDefault("smtp.auth.file", d.SMTP.Auth.File)
	v.SetDefault("smtp.auth.allow_insecure", d.SMTP.Auth.AllowInsecure)
	v.SetDefault("smtp.auth.accept_any", d.SMTP.Auth.AcceptAny)

	v.SetDefault("web.enabled", d.Web.Enabled)
	v.SetDefault("web.host", d.Web.Host)
	v.SetDefault("web.port", d.Web.Port)
	v.SetDefault("web.base_path", d.Web.BasePath)
	v.SetDefault("web.cors_origins", d.Web.CORSOrigins)
	v.SetDefault("web.auth.file", d.Web.Auth.File)
	v.SetDefault("web.tls.cert", d.Web.Tls.Cert)
	v.SetDefault("web.tls.key", d.Web.Tls.Key)

	v.SetDefault("api.host", d.API.Host)
	v.SetDefault("api.port", d.API.Port)
	v.SetDefault("api.base_path", d.API.BasePath)
	v.SetDefault("api.auth.enabled", d.API.Auth.Enabled)
	v.SetDefault("api.auth.api_key", d.API.Auth.APIKey)

	v.SetDefault("storage.driver", d.Storage.Driver)
	v.SetDefault("storage.path", d.Storage.Path)
	v.SetDefault("storage.retention.max_messages", d.Storage.Retention.MaxMessages)
	v.SetDefault("storage.retention.max_age_hours", d.Storage.Retention.MaxAgeHours)
	v.SetDefault("storage.retention.sweep_interval_minutes", d.Storage.Retention.SweepIntervalMinutes)
	v.SetDefault("storage.attachments.store_on_disk", d.Storage.Attachments.StoreOnDisk)
	v.SetDefault("storage.attachments.disk_path", d.Storage.Attachments.DiskPath)

	v.SetDefault("logging.level", d.Logging.Level)
	v.SetDefault("logging.format", d.Logging.Format)
	v.SetDefault("logging.file", d.Logging.File)

	v.SetDefault("server.shutdown_timeout_seconds", d.Server.ShutdownTimeoutSeconds)

	v.SetDefault("shell.command_prefix", d.Shell.CommandPrefix)
	v.SetDefault("shell.prompt", d.Shell.Prompt)
	v.SetDefault("shell.history_file", d.Shell.HistoryFile)
	v.SetDefault("shell.history_size", d.Shell.HistorySize)
	v.SetDefault("shell.color", d.Shell.Color)
	v.SetDefault("shell.seed", d.Shell.Seed)
	v.SetDefault("shell.editor", d.Shell.Editor)
	v.SetDefault("shell.sh_enabled", d.Shell.ShEnabled)
	v.SetDefault("shell.exit_on_error", d.Shell.ExitOnError)
	v.SetDefault("shell.abbr_trigger_key", d.Shell.AbbrTriggerKey)
	v.SetDefault("shell.template_enabled", d.Shell.TemplateEnabled)
	v.SetDefault("shell.template_unsafe_funcs", d.Shell.TemplateUnsafeFuncs)

	v.SetDefault("compose.listen", d.Compose.Listen)
	v.SetDefault("compose.api_addr", d.Compose.APIAddr)
	v.SetDefault("compose.api_user", d.Compose.APIUser)
	v.SetDefault("compose.api_pass", d.Compose.APIPass)
	v.SetDefault("compose.api_insecure_skip_verify", d.Compose.APIInsecureSkipVerify)
	v.SetDefault("compose.api_ca_cert", d.Compose.APICACert)
	v.SetDefault("compose.smtp_addr", d.Compose.SMTPAddr)
	v.SetDefault("compose.smtp_user", d.Compose.SMTPUser)
	v.SetDefault("compose.smtp_pass", d.Compose.SMTPPass)
	v.SetDefault("compose.open", d.Compose.Open)
}

// newEnvReplacer maps YAML key path "smtp.max_message_size_mb" to the env
// var suffix "SMTP_MAX_MESSAGE_SIZE_MB", matching SPEC.md §3.2's
// MAELSINK_<SECTION>_<KEY> convention.
func newEnvReplacer() *strings.Replacer {
	return strings.NewReplacer(".", "_")
}

// bindEnv explicitly binds every key so AutomaticEnv can find nested keys
// (viper's automatic env lookup does not discover keys that were never
// otherwise referenced).
func bindEnv(v *viper.Viper) {
	keys := []string{
		"smtp.host", "smtp.port", "smtp.domain", "smtp.max_message_size_mb",
		"smtp.require_starttls", "smtp.require_tls", "smtp.tls_cert", "smtp.tls_key",
		"smtp.auth.enabled", "smtp.auth.username", "smtp.auth.password",
		"smtp.auth.file", "smtp.auth.allow_insecure", "smtp.auth.accept_any",

		"web.enabled", "web.host", "web.port", "web.base_path", "web.cors_origins", "web.auth.file",
		"web.tls.cert", "web.tls.key",

		"api.host", "api.port", "api.base_path",
		"api.auth.enabled", "api.auth.api_key",

		"storage.driver", "storage.path",
		"storage.retention.max_messages", "storage.retention.max_age_hours",
		"storage.retention.sweep_interval_minutes",
		"storage.attachments.store_on_disk", "storage.attachments.disk_path",

		"logging.level", "logging.format", "logging.file",

		"server.shutdown_timeout_seconds",

		"shell.command_prefix", "shell.prompt", "shell.history_file",
		"shell.history_size", "shell.color", "shell.seed", "shell.editor",
		"shell.sh_enabled", "shell.exit_on_error", "shell.abbr_trigger_key",
		"shell.template_enabled", "shell.template_unsafe_funcs",

		"compose.listen", "compose.api_addr", "compose.api_user", "compose.api_pass",
		"compose.api_insecure_skip_verify", "compose.api_ca_cert",
		"compose.smtp_addr", "compose.smtp_user", "compose.smtp_pass", "compose.open",
	}
	for _, k := range keys {
		_ = v.BindEnv(k)
	}
}

// applyFlagOverrides layers explicitly-set CLI flag values on top of the
// defaults/file/env-resolved config, the highest-precedence layer.
func applyFlagOverrides(cfg *Config, f FlagOverrides) {
	if f.SMTPHost != nil {
		cfg.SMTP.Host = *f.SMTPHost
	}
	if f.SMTPPort != nil {
		cfg.SMTP.Port = *f.SMTPPort
	}
	if f.SMTPDomain != nil {
		cfg.SMTP.Domain = *f.SMTPDomain
	}
	if f.SMTPMaxMessageSizeMB != nil {
		cfg.SMTP.MaxMessageSizeMB = *f.SMTPMaxMessageSizeMB
	}
	if f.SMTPRequireStartTLS != nil {
		cfg.SMTP.RequireStartTLS = *f.SMTPRequireStartTLS
	}
	if f.SMTPRequireTLS != nil {
		cfg.SMTP.RequireTLS = *f.SMTPRequireTLS
	}
	if f.SMTPTLSCert != nil {
		cfg.SMTP.TLSCert = *f.SMTPTLSCert
	}
	if f.SMTPTLSKey != nil {
		cfg.SMTP.TLSKey = *f.SMTPTLSKey
	}
	if f.SMTPAuthEnabled != nil {
		cfg.SMTP.Auth.Enabled = *f.SMTPAuthEnabled
	}
	if f.SMTPAuthUsername != nil {
		cfg.SMTP.Auth.Username = *f.SMTPAuthUsername
	}
	if f.SMTPAuthPassword != nil {
		cfg.SMTP.Auth.Password = *f.SMTPAuthPassword
	}
	if f.SMTPAuthFile != nil {
		cfg.SMTP.Auth.File = *f.SMTPAuthFile
	}
	if f.SMTPAuthAllowInsecure != nil {
		cfg.SMTP.Auth.AllowInsecure = *f.SMTPAuthAllowInsecure
	}
	if f.SMTPAuthAcceptAny != nil {
		cfg.SMTP.Auth.AcceptAny = *f.SMTPAuthAcceptAny
	}
	if f.WebEnabled != nil {
		cfg.Web.Enabled = *f.WebEnabled
	}
	if f.WebHost != nil {
		cfg.Web.Host = *f.WebHost
	}
	if f.WebPort != nil {
		cfg.Web.Port = *f.WebPort
	}
	if f.WebBasePath != nil {
		cfg.Web.BasePath = *f.WebBasePath
	}
	if f.WebCORSOrigins != nil {
		cfg.Web.CORSOrigins = *f.WebCORSOrigins
	}
	if f.WebAuthFile != nil {
		cfg.Web.Auth.File = *f.WebAuthFile
	}
	if f.WebTLSCert != nil {
		cfg.Web.Tls.Cert = *f.WebTLSCert
	}
	if f.WebTLSKey != nil {
		cfg.Web.Tls.Key = *f.WebTLSKey
	}
	if f.APIHost != nil {
		cfg.API.Host = *f.APIHost
	}
	if f.APIPort != nil {
		cfg.API.Port = *f.APIPort
	}
	if f.APIBasePath != nil {
		cfg.API.BasePath = *f.APIBasePath
	}
	if f.APIAuthEnabled != nil {
		cfg.API.Auth.Enabled = *f.APIAuthEnabled
	}
	if f.APIAuthAPIKey != nil {
		cfg.API.Auth.APIKey = *f.APIAuthAPIKey
	}
	if f.DBPath != nil {
		cfg.Storage.Path = *f.DBPath
	}
	if f.StorageDriver != nil {
		cfg.Storage.Driver = *f.StorageDriver
	}
	if f.StorageAttachmentsStoreOnDisk != nil {
		cfg.Storage.Attachments.StoreOnDisk = *f.StorageAttachmentsStoreOnDisk
	}
	if f.StorageAttachmentsDiskPath != nil {
		cfg.Storage.Attachments.DiskPath = *f.StorageAttachmentsDiskPath
	}
	if f.LogLevel != nil {
		cfg.Logging.Level = *f.LogLevel
	}
	if f.LogFormat != nil {
		cfg.Logging.Format = *f.LogFormat
	}
	if f.LogFile != nil {
		cfg.Logging.File = *f.LogFile
	}
	if f.RetentionMaxMessages != nil {
		cfg.Storage.Retention.MaxMessages = *f.RetentionMaxMessages
	}
	if f.RetentionMaxAgeHours != nil {
		cfg.Storage.Retention.MaxAgeHours = *f.RetentionMaxAgeHours
	}
	if f.RetentionSweepIntervalMinutes != nil {
		cfg.Storage.Retention.SweepIntervalMinutes = *f.RetentionSweepIntervalMinutes
	}
	if f.ServerShutdownTimeoutSeconds != nil {
		cfg.Server.ShutdownTimeoutSeconds = *f.ServerShutdownTimeoutSeconds
	}
	if f.ShellCommandPrefix != nil {
		cfg.Shell.CommandPrefix = *f.ShellCommandPrefix
	}
	if f.ShellPrompt != nil {
		cfg.Shell.Prompt = *f.ShellPrompt
	}
	if f.ShellHistoryFile != nil {
		cfg.Shell.HistoryFile = *f.ShellHistoryFile
	}
	if f.ShellHistorySize != nil {
		cfg.Shell.HistorySize = *f.ShellHistorySize
	}
	if f.ShellColor != nil {
		cfg.Shell.Color = *f.ShellColor
	}
	if f.ShellSeed != nil {
		cfg.Shell.Seed = *f.ShellSeed
	}
	if f.ShellEditor != nil {
		cfg.Shell.Editor = *f.ShellEditor
	}
	if f.ShellShEnabled != nil {
		cfg.Shell.ShEnabled = *f.ShellShEnabled
	}
	if f.ShellExitOnError != nil {
		cfg.Shell.ExitOnError = *f.ShellExitOnError
	}
	if f.ShellAbbrTriggerKey != nil {
		cfg.Shell.AbbrTriggerKey = *f.ShellAbbrTriggerKey
	}
	if f.ShellTemplateEnabled != nil {
		cfg.Shell.TemplateEnabled = *f.ShellTemplateEnabled
	}
	if f.ShellTemplateUnsafeFuncs != nil {
		cfg.Shell.TemplateUnsafeFuncs = *f.ShellTemplateUnsafeFuncs
	}
	if f.ComposeListen != nil {
		cfg.Compose.Listen = *f.ComposeListen
	}
	if f.ComposeAPIAddr != nil {
		cfg.Compose.APIAddr = *f.ComposeAPIAddr
	}
	if f.ComposeAPIUser != nil {
		cfg.Compose.APIUser = *f.ComposeAPIUser
	}
	if f.ComposeAPIPass != nil {
		cfg.Compose.APIPass = *f.ComposeAPIPass
	}
	if f.ComposeAPIInsecureSkipVerify != nil {
		cfg.Compose.APIInsecureSkipVerify = *f.ComposeAPIInsecureSkipVerify
	}
	if f.ComposeAPICACert != nil {
		cfg.Compose.APICACert = *f.ComposeAPICACert
	}
	if f.ComposeSMTPAddr != nil {
		cfg.Compose.SMTPAddr = *f.ComposeSMTPAddr
	}
	if f.ComposeSMTPUser != nil {
		cfg.Compose.SMTPUser = *f.ComposeSMTPUser
	}
	if f.ComposeSMTPPass != nil {
		cfg.Compose.SMTPPass = *f.ComposeSMTPPass
	}
	if f.ComposeOpen != nil {
		cfg.Compose.Open = *f.ComposeOpen
	}
}
