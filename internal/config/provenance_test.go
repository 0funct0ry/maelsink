package config

import (
	"os"
	"path/filepath"
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
