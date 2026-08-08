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
	v.SetDefault("smtp.starttls", d.SMTP.StartTLS)
	v.SetDefault("smtp.tls_cert", d.SMTP.TLSCert)
	v.SetDefault("smtp.tls_key", d.SMTP.TLSKey)
	v.SetDefault("smtp.auth.enabled", d.SMTP.Auth.Enabled)
	v.SetDefault("smtp.auth.username", d.SMTP.Auth.Username)
	v.SetDefault("smtp.auth.password", d.SMTP.Auth.Password)

	v.SetDefault("web.enabled", d.Web.Enabled)
	v.SetDefault("web.host", d.Web.Host)
	v.SetDefault("web.port", d.Web.Port)
	v.SetDefault("web.base_path", d.Web.BasePath)
	v.SetDefault("web.cors_origins", d.Web.CORSOrigins)

	v.SetDefault("api.host", d.API.Host)
	v.SetDefault("api.port", d.API.Port)
	v.SetDefault("api.base_path", d.API.BasePath)
	v.SetDefault("api.auth.enabled", d.API.Auth.Enabled)
	v.SetDefault("api.auth.api_key", d.API.Auth.APIKey)

	v.SetDefault("storage.driver", d.Storage.Driver)
	v.SetDefault("storage.path", d.Storage.Path)
	v.SetDefault("storage.retention.max_messages", d.Storage.Retention.MaxMessages)
	v.SetDefault("storage.retention.max_age_hours", d.Storage.Retention.MaxAgeHours)
	v.SetDefault("storage.attachments.store_on_disk", d.Storage.Attachments.StoreOnDisk)
	v.SetDefault("storage.attachments.disk_path", d.Storage.Attachments.DiskPath)

	v.SetDefault("logging.level", d.Logging.Level)
	v.SetDefault("logging.format", d.Logging.Format)
	v.SetDefault("logging.file", d.Logging.File)

	v.SetDefault("server.shutdown_timeout_seconds", d.Server.ShutdownTimeoutSeconds)
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
		"smtp.starttls", "smtp.tls_cert", "smtp.tls_key",
		"smtp.auth.enabled", "smtp.auth.username", "smtp.auth.password",

		"web.enabled", "web.host", "web.port", "web.base_path", "web.cors_origins",

		"api.host", "api.port", "api.base_path",
		"api.auth.enabled", "api.auth.api_key",

		"storage.driver", "storage.path",
		"storage.retention.max_messages", "storage.retention.max_age_hours",
		"storage.attachments.store_on_disk", "storage.attachments.disk_path",

		"logging.level", "logging.format", "logging.file",

		"server.shutdown_timeout_seconds",
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
	if f.SMTPStartTLS != nil {
		cfg.SMTP.StartTLS = *f.SMTPStartTLS
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
	if f.ServerShutdownTimeoutSeconds != nil {
		cfg.Server.ShutdownTimeoutSeconds = *f.ServerShutdownTimeoutSeconds
	}
}
