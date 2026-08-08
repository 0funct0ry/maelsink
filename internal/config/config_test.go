package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad_DefaultsOnly(t *testing.T) {
	dir := t.TempDir()
	oldwd, _ := os.Getwd()
	defer os.Chdir(oldwd)
	os.Chdir(dir)

	cfg, err := Load(Options{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.SMTP.Port != 1025 {
		t.Errorf("smtp.port = %d, want default 1025", cfg.SMTP.Port)
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("logging.format = %q, want default text", cfg.Logging.Format)
	}
}

func TestLoad_FileOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "maelsink.yaml")
	if err := os.WriteFile(path, []byte("smtp:\n  port: 2525\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(Options{ConfigFile: path})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.SMTP.Port != 2525 {
		t.Errorf("smtp.port = %d, want file override 2525", cfg.SMTP.Port)
	}
	// Untouched keys still fall back to defaults.
	if cfg.SMTP.Domain != "maelsink.local" {
		t.Errorf("smtp.domain = %q, want default", cfg.SMTP.Domain)
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "maelsink.yaml")
	if err := os.WriteFile(path, []byte("smtp:\n  port: 2525\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("MAELSINK_SMTP_PORT", "3025")

	cfg, err := Load(Options{ConfigFile: path})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.SMTP.Port != 3025 {
		t.Errorf("smtp.port = %d, want env override 3025", cfg.SMTP.Port)
	}
}

func TestLoad_FlagOverridesEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "maelsink.yaml")
	if err := os.WriteFile(path, []byte("smtp:\n  port: 2525\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("MAELSINK_SMTP_PORT", "3025")

	flagPort := 4025
	cfg, err := Load(Options{
		ConfigFile: path,
		Flags:      FlagOverrides{SMTPPort: &flagPort},
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.SMTP.Port != 4025 {
		t.Errorf("smtp.port = %d, want flag override 4025", cfg.SMTP.Port)
	}
}

func TestDefaults_YAMLRoundTrip(t *testing.T) {
	d := Defaults()
	data, err := d.YAML()
	if err != nil {
		t.Fatalf("YAML: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "maelsink.yaml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Load(Options{ConfigFile: path})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, d) {
		t.Errorf("round-tripped config = %+v, want %+v", got, d)
	}
}

func TestValidate(t *testing.T) {
	cfg := Defaults()
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() on defaults = %v, want nil", err)
	}

	bad := Defaults()
	bad.SMTP.Port = 0
	if err := bad.Validate(); err == nil {
		t.Errorf("Validate() with invalid port = nil, want error")
	}
}

func TestValidate_SMTPMaxMessageSize(t *testing.T) {
	cfg := Defaults()
	cfg.SMTP.MaxMessageSizeMB = 0
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() with max_message_size_mb=0 = nil, want error")
	}
}

func TestValidate_SMTPStartTLSRequiresCertAndKey(t *testing.T) {
	cfg := Defaults()
	cfg.SMTP.StartTLS = true
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() with starttls enabled and no cert/key = nil, want error")
	}

	cfg.SMTP.TLSCert = "cert.pem"
	cfg.SMTP.TLSKey = "key.pem"
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() with starttls enabled and cert/key set = %v, want nil", err)
	}
}

func TestValidate_SMTPAuthRequiresUsernameAndPassword(t *testing.T) {
	cfg := Defaults()
	cfg.SMTP.Auth.Enabled = true
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() with auth enabled and no credentials = nil, want error")
	}

	cfg.SMTP.Auth.Username = "user"
	cfg.SMTP.Auth.Password = "pass"
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() with auth enabled and credentials set = %v, want nil", err)
	}
}

func TestWriteFile_RefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "maelsink.yaml")

	if err := WriteFile(path, Defaults(), false); err != nil {
		t.Fatalf("first WriteFile: %v", err)
	}
	if err := WriteFile(path, Defaults(), false); err == nil {
		t.Errorf("second WriteFile without force = nil, want error")
	}
	if err := WriteFile(path, Defaults(), true); err != nil {
		t.Errorf("WriteFile with force = %v, want nil", err)
	}
}
