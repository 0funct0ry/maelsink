package config

// Entry is one row of the Settings screen's full config dump: a single
// non-secret key, its resolved value, and the Source that produced it.
type Entry struct {
	Section string `json:"section"`
	Key     string `json:"key"`
	Value   any    `json:"value"`
	Source  Source `json:"source"`
}

// Dump returns every non-secret config key's resolved value plus its
// Provenance entry, grouped by section (smtp/web/api/storage/logging/
// server), for GET /ui-api/v1/config. smtp.auth.password and
// api.auth.api_key are never included, matching keyToFlag's exclusion.
func Dump(cfg Config, prov Provenance) []Entry {
	entries := make([]Entry, 0, len(keyToFlag))
	add := func(section, key string, value any) {
		entries = append(entries, Entry{Section: section, Key: key, Value: value, Source: prov[key]})
	}

	add("smtp", "smtp.host", cfg.SMTP.Host)
	add("smtp", "smtp.port", cfg.SMTP.Port)
	add("smtp", "smtp.domain", cfg.SMTP.Domain)
	add("smtp", "smtp.max_message_size_mb", cfg.SMTP.MaxMessageSizeMB)
	add("smtp", "smtp.starttls", cfg.SMTP.StartTLS)
	add("smtp", "smtp.tls_cert", cfg.SMTP.TLSCert)
	add("smtp", "smtp.tls_key", cfg.SMTP.TLSKey)
	add("smtp", "smtp.auth.enabled", cfg.SMTP.Auth.Enabled)
	add("smtp", "smtp.auth.username", cfg.SMTP.Auth.Username)

	add("web", "web.enabled", cfg.Web.Enabled)
	add("web", "web.host", cfg.Web.Host)
	add("web", "web.port", cfg.Web.Port)
	add("web", "web.base_path", cfg.Web.BasePath)
	add("web", "web.cors_origins", cfg.Web.CORSOrigins)

	add("api", "api.host", cfg.API.Host)
	add("api", "api.port", cfg.API.Port)
	add("api", "api.base_path", cfg.API.BasePath)
	add("api", "api.auth.enabled", cfg.API.Auth.Enabled)

	add("storage", "storage.driver", cfg.Storage.Driver)
	add("storage", "storage.path", cfg.Storage.Path)
	add("storage", "storage.retention.max_messages", cfg.Storage.Retention.MaxMessages)
	add("storage", "storage.retention.max_age_hours", cfg.Storage.Retention.MaxAgeHours)
	add("storage", "storage.retention.sweep_interval_minutes", cfg.Storage.Retention.SweepIntervalMinutes)
	add("storage", "storage.attachments.store_on_disk", cfg.Storage.Attachments.StoreOnDisk)
	add("storage", "storage.attachments.disk_path", cfg.Storage.Attachments.DiskPath)

	add("logging", "logging.level", cfg.Logging.Level)
	add("logging", "logging.format", cfg.Logging.Format)
	add("logging", "logging.file", cfg.Logging.File)

	add("server", "server.shutdown_timeout_seconds", cfg.Server.ShutdownTimeoutSeconds)

	return entries
}
