package config

import "path/filepath"

// Entry is one row of the Settings screen's full config dump: a single
// key, its resolved value, and the Source that produced it. When Secret is
// true, Value is never the actual configured value (it's nil, or a bool
// indicating only whether something is set) — the Settings screen must show
// that a secret is configured and by which layer, but never its contents.
type Entry struct {
	Section string `json:"section"`
	Key     string `json:"key"`
	Value   any    `json:"value"`
	Secret  bool   `json:"secret"`
	Source  Source `json:"source"`
}

// Dump returns every config key's resolved value (or, for secret keys, only
// whether one is set) plus its Provenance entry, grouped by section
// (smtp/web/api/storage/logging/server), for GET /ui-api/v1/config.
// smtp.auth.password and api.auth.api_key are included via addSecret so
// their provenance is visible, but the actual secret value never leaves the
// server (see addSecret and ResolveProvenance's flag-origin masking).
func Dump(cfg Config, prov Provenance) []Entry {
	entries := make([]Entry, 0, len(keyToFlag))
	add := func(section, key string, value any) {
		entries = append(entries, Entry{Section: section, Key: key, Value: value, Source: prov[key]})
	}
	addSecret := func(section, key string, isSet bool) {
		entries = append(entries, Entry{Section: section, Key: key, Value: isSet, Secret: true, Source: prov[key]})
	}

	add("smtp", "smtp.host", cfg.SMTP.Host)
	add("smtp", "smtp.port", cfg.SMTP.Port)
	add("smtp", "smtp.domain", cfg.SMTP.Domain)
	add("smtp", "smtp.max_message_size_mb", cfg.SMTP.MaxMessageSizeMB)
	add("smtp", "smtp.require_starttls", cfg.SMTP.RequireStartTLS)
	add("smtp", "smtp.require_tls", cfg.SMTP.RequireTLS)
	add("smtp", "smtp.tls_cert", cfg.SMTP.TLSCert)
	add("smtp", "smtp.tls_key", cfg.SMTP.TLSKey)
	add("smtp", "smtp.auth.enabled", cfg.SMTP.Auth.Enabled)
	add("smtp", "smtp.auth.username", cfg.SMTP.Auth.Username)
	addSecret("smtp", "smtp.auth.password", cfg.SMTP.Auth.Password != "")
	add("smtp", "smtp.auth.file", basename(cfg.SMTP.Auth.File))
	add("smtp", "smtp.auth.allow_insecure", cfg.SMTP.Auth.AllowInsecure)
	add("smtp", "smtp.auth.accept_any", cfg.SMTP.Auth.AcceptAny)

	add("web", "web.enabled", cfg.Web.Enabled)
	add("web", "web.host", cfg.Web.Host)
	add("web", "web.port", cfg.Web.Port)
	add("web", "web.base_path", cfg.Web.BasePath)
	add("web", "web.cors_origins", cfg.Web.CORSOrigins)
	// web.auth.file/tls.cert/tls.key are reduced to their basename, same as
	// storage.path below (M8.7 abstraction-leakage hardening) — these are
	// file paths, not secrets, but the server's directory layout shouldn't
	// leak to the client either.
	add("web", "web.auth.file", basename(cfg.Web.Auth.File))
	add("web", "web.tls.cert", basename(cfg.Web.Tls.Cert))
	add("web", "web.tls.key", basename(cfg.Web.Tls.Key))

	add("api", "api.host", cfg.API.Host)
	add("api", "api.port", cfg.API.Port)
	add("api", "api.base_path", cfg.API.BasePath)
	add("api", "api.auth.enabled", cfg.API.Auth.Enabled)
	addSecret("api", "api.auth.api_key", cfg.API.Auth.APIKey != "")

	add("storage", "storage.driver", cfg.Storage.Driver)
	// storage.path/disk_path are reduced to their basename — the Settings
	// screen's value is showing *that* a path is configured, not the
	// server's directory layout or OS username (M8.7 abstraction-leakage
	// hardening).
	add("storage", "storage.path", basename(cfg.Storage.Path))
	add("storage", "storage.retention.max_messages", cfg.Storage.Retention.MaxMessages)
	add("storage", "storage.retention.max_age_hours", cfg.Storage.Retention.MaxAgeHours)
	add("storage", "storage.retention.sweep_interval_minutes", cfg.Storage.Retention.SweepIntervalMinutes)
	add("storage", "storage.attachments.store_on_disk", cfg.Storage.Attachments.StoreOnDisk)
	add("storage", "storage.attachments.disk_path", basename(cfg.Storage.Attachments.DiskPath))

	add("logging", "logging.level", cfg.Logging.Level)
	add("logging", "logging.format", cfg.Logging.Format)
	add("logging", "logging.file", cfg.Logging.File)

	add("server", "server.shutdown_timeout_seconds", cfg.Server.ShutdownTimeoutSeconds)

	return entries
}

// basename reduces a filesystem path to its final element, leaving an empty
// path empty rather than collapsing it to filepath.Base's "." for the empty
// string.
func basename(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Base(path)
}
