package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func TestDump_ExcludesSecrets(t *testing.T) {
	cfg := Defaults()
	cfg.SMTP.Auth.Password = "supersecret"
	cfg.API.Auth.APIKey = "topsecretkey"

	entries := Dump(cfg, Provenance{})

	for _, e := range entries {
		if e.Key == "smtp.auth.password" || e.Key == "api.auth.api_key" {
			t.Fatalf("Dump() included secret key %q", e.Key)
		}
		if s, ok := e.Value.(string); ok {
			if s == "supersecret" || s == "topsecretkey" {
				t.Fatalf("Dump() leaked a secret value via key %q", e.Key)
			}
		}
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

func TestResolveProvenance_NeverIncludesSecretKeys(t *testing.T) {
	for _, k := range ProvenanceKeys() {
		if k == "smtp.auth.password" || k == "api.auth.api_key" {
			t.Fatalf("ProvenanceKeys() included secret key %q", k)
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
