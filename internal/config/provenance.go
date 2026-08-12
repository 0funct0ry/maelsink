package config

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Source describes which configuration layer resolved a given key and, for
// every layer but "default", the specific origin within that layer (the
// flag string, the env var name, or the config file path) — SPEC.md §8.1's
// Settings screen "source badge" + "origin" column.
//
// Origin is intentionally file-path-only for the "file" layer: line-number
// provenance (e.g. "maelsink.yaml:3") would require a YAML parser that
// preserves line info (gopkg.in/yaml.v3's yaml.Node) and is a materially
// bigger change, out of scope for this milestone.
type Source struct {
	Layer  string `json:"layer"` // "default" | "file" | "env" | "flag"
	Origin string `json:"origin"`
}

// Provenance maps a config key (dotted path, e.g. "smtp.host") to the
// Source that resolved it.
type Provenance map[string]Source

// keyToFlag maps every non-secret config key eligible for the Settings
// screen's provenance table (smtp/web/api/storage/logging/server sections
// only — shell.* keys aren't part of that screen) to the CLI flag name that
// can override it. smtp.auth.password and api.auth.api_key are
// deliberately absent: SPEC.md's "non-secret fields" requirement means
// secret values are never resolved or surfaced here at all.
var keyToFlag = map[string]string{
	"smtp.host":                       "smtp-host",
	"smtp.port":                       "smtp-port",
	"smtp.domain":                     "smtp-domain",
	"smtp.max_message_size_mb":        "smtp-max-message-size-mb",
	"smtp.starttls":                   "smtp-starttls",
	"smtp.tls_cert":                   "smtp-tls-cert",
	"smtp.tls_key":                    "smtp-tls-key",
	"smtp.auth.enabled":               "smtp-auth-enabled",
	"smtp.auth.username":              "smtp-auth-username",
	"web.enabled":                     "web-enabled",
	"web.host":                        "web-host",
	"web.port":                        "web-port",
	"web.base_path":                   "web-base-path",
	"web.cors_origins":                "web-cors-origins",
	"api.host":                        "api-host",
	"api.port":                        "api-port",
	"api.base_path":                   "api-base-path",
	"api.auth.enabled":                "api-auth-enabled",
	"storage.driver":                  "storage-driver",
	"storage.path":                    "db",
	"storage.retention.max_messages":  "retention-max-messages",
	"storage.retention.max_age_hours": "retention-max-age-hours",
	"storage.retention.sweep_interval_minutes": "retention-sweep-interval-minutes",
	"storage.attachments.store_on_disk":        "storage-attachments-store-on-disk",
	"storage.attachments.disk_path":            "storage-attachments-disk-path",
	"logging.level":                            "log-level",
	"logging.format":                           "log-format",
	"logging.file":                             "log-file",
	"server.shutdown_timeout_seconds":          "server-shutdown-timeout-seconds",
}

// isPathKey reports whether key's resolved value is a filesystem path, so
// its provenance origin (e.g. a --flag=value string) is reduced to a
// basename the same way Dump reduces the value itself (M8.7
// abstraction-leakage hardening).
func isPathKey(key string) bool {
	return key == "storage.path" || key == "storage.attachments.disk_path"
}

// ProvenanceKeys returns every key ResolveProvenance/Dump know about, sorted
// for determinism.
func ProvenanceKeys() []string {
	keys := make([]string, 0, len(keyToFlag))
	for k := range keyToFlag {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ResolveProvenance reports, for every key in envKeys, which layer resolved
// it and that layer's origin, checking in the same precedence order
// applyFlagOverrides/resolveConfig already encode: flag > env > file >
// default.
//
//   - flag: flagSet.Changed(flagName) is true. Origin is "--flagName=value".
//   - env: the matching MAELSINK_<KEY> env var is set in os.Environ().
//     Origin is the env var name.
//   - file: v.InConfig(key) — viper's own "was this key present in the
//     loaded config file" check. Origin is the resolved config file's
//     basename only (M8.7 abstraction-leakage hardening) — the Settings
//     screen's value is showing *which* config file set this key, not the
//     server's directory layout or OS username.
//   - default: none of the above matched.
func ResolveProvenance(v *viper.Viper, flagSet *pflag.FlagSet, envKeys []string) Provenance {
	prov := make(Provenance, len(envKeys))
	replacer := newEnvReplacer()

	for _, key := range envKeys {
		if flagSet != nil {
			if flagName, ok := keyToFlag[key]; ok {
				if f := flagSet.Lookup(flagName); f != nil && f.Changed {
					value := f.Value.String()
					if isPathKey(key) {
						value = filepath.Base(value)
					}
					prov[key] = Source{Layer: "flag", Origin: "--" + flagName + "=" + value}
					continue
				}
			}
		}

		envVar := "MAELSINK_" + strings.ToUpper(replacer.Replace(key))
		if _, ok := os.LookupEnv(envVar); ok {
			prov[key] = Source{Layer: "env", Origin: envVar}
			continue
		}

		if v != nil && v.InConfig(key) {
			prov[key] = Source{Layer: "file", Origin: filepath.Base(v.ConfigFileUsed())}
			continue
		}

		prov[key] = Source{Layer: "default", Origin: ""}
	}

	return prov
}
