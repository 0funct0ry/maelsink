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

func TestLoad_ShellKeysReachableFromFileEnvFlag(t *testing.T) {
	cases := []struct {
		name     string
		yamlKey  string
		yamlVal  string
		envKey   string
		envVal   string
		flags    func(v string) FlagOverrides
		flagVal  string
		get      func(c Config) any
		wantFile any
		wantEnv  any
		wantFlag any
	}{
		{
			name:    "command_prefix",
			yamlKey: "command_prefix", yamlVal: "!",
			envKey: "MAELSINK_SHELL_COMMAND_PREFIX", envVal: "@",
			flagVal:  "#",
			flags:    func(v string) FlagOverrides { return FlagOverrides{ShellCommandPrefix: &v} },
			get:      func(c Config) any { return c.Shell.CommandPrefix },
			wantFile: "!", wantEnv: "@", wantFlag: "#",
		},
		{
			name:    "prompt",
			yamlKey: "prompt", yamlVal: "file> ",
			envKey: "MAELSINK_SHELL_PROMPT", envVal: "env> ",
			flagVal:  "flag> ",
			flags:    func(v string) FlagOverrides { return FlagOverrides{ShellPrompt: &v} },
			get:      func(c Config) any { return c.Shell.Prompt },
			wantFile: "file> ", wantEnv: "env> ", wantFlag: "flag> ",
		},
		{
			name:    "history_file",
			yamlKey: "history_file", yamlVal: "/tmp/file_hist",
			envKey: "MAELSINK_SHELL_HISTORY_FILE", envVal: "/tmp/env_hist",
			flagVal:  "/tmp/flag_hist",
			flags:    func(v string) FlagOverrides { return FlagOverrides{ShellHistoryFile: &v} },
			get:      func(c Config) any { return c.Shell.HistoryFile },
			wantFile: "/tmp/file_hist", wantEnv: "/tmp/env_hist", wantFlag: "/tmp/flag_hist",
		},
		{
			name:    "history_size",
			yamlKey: "history_size", yamlVal: "111",
			envKey: "MAELSINK_SHELL_HISTORY_SIZE", envVal: "222",
			flagVal: "333",
			flags: func(v string) FlagOverrides {
				n := 333
				return FlagOverrides{ShellHistorySize: &n}
			},
			get:      func(c Config) any { return c.Shell.HistorySize },
			wantFile: 111, wantEnv: 222, wantFlag: 333,
		},
		{
			name:    "color",
			yamlKey: "color", yamlVal: "always",
			envKey: "MAELSINK_SHELL_COLOR", envVal: "never",
			flagVal:  "auto",
			flags:    func(v string) FlagOverrides { return FlagOverrides{ShellColor: &v} },
			get:      func(c Config) any { return c.Shell.Color },
			wantFile: "always", wantEnv: "never", wantFlag: "auto",
		},
		{
			name:    "editor",
			yamlKey: "editor", yamlVal: "vim",
			envKey: "MAELSINK_SHELL_EDITOR", envVal: "nano",
			flagVal:  "emacs",
			flags:    func(v string) FlagOverrides { return FlagOverrides{ShellEditor: &v} },
			get:      func(c Config) any { return c.Shell.Editor },
			wantFile: "vim", wantEnv: "nano", wantFlag: "emacs",
		},
		{
			name:    "abbr_trigger_key",
			yamlKey: "abbr_trigger_key", yamlVal: "tab",
			envKey: "MAELSINK_SHELL_ABBR_TRIGGER_KEY", envVal: "enter",
			flagVal:  "none",
			flags:    func(v string) FlagOverrides { return FlagOverrides{ShellAbbrTriggerKey: &v} },
			get:      func(c Config) any { return c.Shell.AbbrTriggerKey },
			wantFile: "tab", wantEnv: "enter", wantFlag: "none",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/file", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "maelsink.yaml")
			data := "shell:\n  " + tc.yamlKey + ": \"" + tc.yamlVal + "\"\n"
			if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(Options{ConfigFile: path})
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got := tc.get(cfg); got != tc.wantFile {
				t.Errorf("shell.%s from file = %v, want %v", tc.yamlKey, got, tc.wantFile)
			}
		})

		t.Run(tc.name+"/env", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "maelsink.yaml")
			data := "shell:\n  " + tc.yamlKey + ": \"" + tc.yamlVal + "\"\n"
			if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Setenv(tc.envKey, tc.envVal)
			cfg, err := Load(Options{ConfigFile: path})
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got := tc.get(cfg); got != tc.wantEnv {
				t.Errorf("shell.%s from env = %v, want %v", tc.yamlKey, got, tc.wantEnv)
			}
		})

		t.Run(tc.name+"/flag", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "maelsink.yaml")
			data := "shell:\n  " + tc.yamlKey + ": \"" + tc.yamlVal + "\"\n"
			if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Setenv(tc.envKey, tc.envVal)
			cfg, err := Load(Options{ConfigFile: path, Flags: tc.flags(tc.flagVal)})
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got := tc.get(cfg); got != tc.wantFlag {
				t.Errorf("shell.%s from flag = %v, want %v", tc.yamlKey, got, tc.wantFlag)
			}
		})
	}
}

func TestLoad_ShellBoolAndInt64KeysReachable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "maelsink.yaml")
	data := "shell:\n  sh_enabled: false\n  exit_on_error: false\n  template_enabled: false\n  template_unsafe_funcs: false\n  seed: 111\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(Options{ConfigFile: path})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Shell.ShEnabled != false || cfg.Shell.ExitOnError != false ||
		cfg.Shell.TemplateEnabled != false || cfg.Shell.TemplateUnsafeFuncs != false ||
		cfg.Shell.Seed != 111 {
		t.Errorf("shell bool/int64 keys from file = %+v, want all false/111", cfg.Shell)
	}

	t.Setenv("MAELSINK_SHELL_SH_ENABLED", "true")
	t.Setenv("MAELSINK_SHELL_EXIT_ON_ERROR", "true")
	t.Setenv("MAELSINK_SHELL_TEMPLATE_ENABLED", "true")
	t.Setenv("MAELSINK_SHELL_TEMPLATE_UNSAFE_FUNCS", "true")
	t.Setenv("MAELSINK_SHELL_SEED", "222")

	cfg, err = Load(Options{ConfigFile: path})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Shell.ShEnabled || !cfg.Shell.ExitOnError ||
		!cfg.Shell.TemplateEnabled || !cfg.Shell.TemplateUnsafeFuncs ||
		cfg.Shell.Seed != 222 {
		t.Errorf("shell bool/int64 keys from env = %+v, want all true/222", cfg.Shell)
	}

	shEnabled, exitOnErr, tmplEnabled, tmplUnsafe := false, false, false, false
	var seed int64 = 333
	cfg, err = Load(Options{ConfigFile: path, Flags: FlagOverrides{
		ShellShEnabled:           &shEnabled,
		ShellExitOnError:         &exitOnErr,
		ShellTemplateEnabled:     &tmplEnabled,
		ShellTemplateUnsafeFuncs: &tmplUnsafe,
		ShellSeed:                &seed,
	}})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Shell.ShEnabled || cfg.Shell.ExitOnError ||
		cfg.Shell.TemplateEnabled || cfg.Shell.TemplateUnsafeFuncs ||
		cfg.Shell.Seed != 333 {
		t.Errorf("shell bool/int64 keys from flag = %+v, want all false/333", cfg.Shell)
	}
}

func TestValidate_ShellColorAndAbbrTriggerKey(t *testing.T) {
	cfg := Defaults()
	cfg.Shell.Color = "bogus"
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() with invalid shell.color = nil, want error")
	}

	cfg = Defaults()
	cfg.Shell.AbbrTriggerKey = "bogus"
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() with invalid shell.abbr_trigger_key = nil, want error")
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
