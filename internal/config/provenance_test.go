package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

// TestDump_MasksSecretValues asserts that secret keys (smtp.auth.password,
// api.auth.api_key) are present in the dump — so the Settings screen can
// show that they're configured and by which layer — but their Value is
// always a "is it set" bool, never the actual secret string, and Secret is
// always true.
func TestDump_MasksSecretValues(t *testing.T) {
	cfg := Defaults()
	cfg.SMTP.Auth.Password = "supersecret"
	cfg.API.Auth.APIKey = "topsecretkey"

	entries := Dump(cfg, Provenance{})

	seen := map[string]bool{}
	for _, e := range entries {
		if e.Key != "smtp.auth.password" && e.Key != "api.auth.api_key" {
			if s, ok := e.Value.(string); ok {
				if s == "supersecret" || s == "topsecretkey" {
					t.Fatalf("Dump() leaked a secret value via non-secret key %q", e.Key)
				}
			}
			continue
		}
		seen[e.Key] = true
		if !e.Secret {
			t.Errorf("Dump() entry %q: Secret = false, want true", e.Key)
		}
		if _, ok := e.Value.(bool); !ok {
			t.Errorf("Dump() entry %q: Value = %v (%T), want a bool", e.Key, e.Value, e.Value)
		}
		if e.Value != true {
			t.Errorf("Dump() entry %q: Value = %v, want true (password/key is set)", e.Key, e.Value)
		}
	}
	if !seen["smtp.auth.password"] || !seen["api.auth.api_key"] {
		t.Fatalf("Dump() did not include both secret keys, got entries: %+v", entries)
	}
}

func TestResolveProvenance_AllFourLayers(t *testing.T) {
	dir := t.TempDir()
	oldwd, _ := os.Getwd()
	defer os.Chdir(oldwd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// File layer: smtp.domain set via maelsink.yaml.
	cfgPath := filepath.Join(dir, "maelsink.yaml")
	if err := os.WriteFile(cfgPath, []byte("smtp:\n  domain: from-file.local\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Env layer: web.host set via MAELSINK_WEB_HOST.
	t.Setenv("MAELSINK_WEB_HOST", "0.0.0.0")

	// Flag layer: api.port set via a --api-port flag.
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	apiPort := fs.Int("api-port", 9090, "")
	if err := fs.Set("api-port", "9999"); err != nil {
		t.Fatal(err)
	}
	_ = apiPort

	var overrides FlagOverrides
	port := 9999
	overrides.APIPort = &port

	cfg, prov, err := LoadWithProvenance(Options{Flags: overrides}, fs)
	if err != nil {
		t.Fatalf("LoadWithProvenance: %v", err)
	}
	if cfg.API.Port != 9999 {
		t.Fatalf("cfg.API.Port = %d, want 9999", cfg.API.Port)
	}

	if got := prov["smtp.domain"]; got.Layer != "file" {
		t.Errorf("smtp.domain provenance = %+v, want layer=file", got)
	}
	if got := prov["web.host"]; got.Layer != "env" || got.Origin != "MAELSINK_WEB_HOST" {
		t.Errorf("web.host provenance = %+v, want layer=env origin=MAELSINK_WEB_HOST", got)
	}
	if got := prov["api.port"]; got.Layer != "flag" || got.Origin != "--api-port=9999" {
		t.Errorf("api.port provenance = %+v, want layer=flag origin=--api-port=9999", got)
	}
	if got := prov["smtp.host"]; got.Layer != "default" || got.Origin != "" {
		t.Errorf("smtp.host provenance = %+v, want layer=default origin=\"\"", got)
	}
}

// TestResolveProvenance_SecretFlagOriginNeverLeaksValue asserts that when a
// secret key is set via a CLI flag, the resulting Source.Origin string
// (e.g. "--smtp-auth-password=...") never embeds the actual flag value.
func TestResolveProvenance_SecretFlagOriginNeverLeaksValue(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("smtp-auth-password", "", "")
	fs.String("api-auth-api-key", "", "")
	if err := fs.Set("smtp-auth-password", "supersecret"); err != nil {
		t.Fatal(err)
	}
	if err := fs.Set("api-auth-api-key", "topsecretkey"); err != nil {
		t.Fatal(err)
	}

	prov := ResolveProvenance(nil, fs, ProvenanceKeys())

	for _, key := range []string{"smtp.auth.password", "api.auth.api_key"} {
		src := prov[key]
		if src.Layer != "flag" {
			t.Fatalf("%s: Layer = %q, want flag", key, src.Layer)
		}
		if strings.Contains(src.Origin, "supersecret") || strings.Contains(src.Origin, "topsecretkey") {
			t.Fatalf("%s: Origin = %q leaked the secret value", key, src.Origin)
		}
	}
}

// TestPathValuesAndOrigins_NeverLeakDirectoryLayout asserts storage.path,
// storage.attachments.disk_path, and the config-file origin string are
// always reduced to a basename — the Settings screen must show *that* a
// path is configured and *what layer* set it, never the server's directory
// layout or OS username (M8.7 abstraction-leakage hardening).
func TestPathValuesAndOrigins_NeverLeakDirectoryLayout(t *testing.T) {
	dir := t.TempDir()
	oldwd, _ := os.Getwd()
	defer os.Chdir(oldwd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(dir, "maelsink.yaml")
	if err := os.WriteFile(cfgPath, []byte("storage:\n  attachments:\n    disk_path: "+filepath.Join(dir, "nested", "attachments")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	dbPath := fs.String("db", "", "")
	dbAbs := filepath.Join(dir, "deep", "nested", "mail.db")
	if err := fs.Set("db", dbAbs); err != nil {
		t.Fatal(err)
	}
	_ = dbPath

	var overrides FlagOverrides
	overrides.DBPath = &dbAbs

	cfg, prov, err := LoadWithProvenance(Options{Flags: overrides}, fs)
	if err != nil {
		t.Fatalf("LoadWithProvenance: %v", err)
	}

	assertNoSeparator := func(label, s string) {
		if strings.ContainsRune(s, os.PathSeparator) || strings.Contains(s, "/") {
			t.Errorf("%s contains a path separator: %q", label, s)
		}
	}

	entries := Dump(cfg, prov)
	for _, e := range entries {
		if e.Key == "storage.path" || e.Key == "storage.attachments.disk_path" {
			if s, ok := e.Value.(string); ok {
				assertNoSeparator("Dump entry "+e.Key+" value", s)
			}
		}
	}

	assertNoSeparator("storage.path origin", prov["storage.path"].Origin)
	assertNoSeparator("storage.attachments.disk_path origin", prov["storage.attachments.disk_path"].Origin)

	if prov["storage.path"].Layer != "flag" {
		t.Fatalf("storage.path provenance = %+v, want layer=flag", prov["storage.path"])
	}
	if prov["storage.attachments.disk_path"].Layer != "file" {
		t.Fatalf("storage.attachments.disk_path provenance = %+v, want layer=file", prov["storage.attachments.disk_path"])
	}
}
