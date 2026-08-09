package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfig_GetSet(t *testing.T) {
	s, out, _ := newTestSession(t, nil)

	if err := (Config{}).Run(context.Background(), s, []string{"set", "prompt", "myshell> "}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if s.Cfg.Prompt != "myshell> " {
		t.Errorf("prompt = %q", s.Cfg.Prompt)
	}

	out.Reset()
	if err := (Config{}).Run(context.Background(), s, []string{"get", "prompt"}); err != nil {
		t.Fatalf("get: %v", err)
	}
	if strings.TrimRight(out.String(), "\n") != "myshell> " {
		t.Errorf("out = %q", out.String())
	}
}

func TestConfig_SaveNewFileRequiresForce(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	dir := t.TempDir()
	path := filepath.Join(dir, "maelsink.yaml")

	if err := (Config{}).Run(context.Background(), s, []string{"save", path}); err == nil {
		t.Fatal("expected error creating a new file without --force")
	}
	if err := (Config{}).Run(context.Background(), s, []string{"save", path, "--force"}); err != nil {
		t.Fatalf("save --force: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "shell:") {
		t.Errorf("data = %s", data)
	}
}

func TestConfig_SavePreservesOtherSections(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	dir := t.TempDir()
	path := filepath.Join(dir, "maelsink.yaml")
	if err := os.WriteFile(path, []byte("smtp:\n  port: 1025\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := (Config{}).Run(context.Background(), s, []string{"save", path}); err != nil {
		t.Fatalf("save: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "smtp:") || !strings.Contains(string(data), "shell:") {
		t.Errorf("data = %s", data)
	}
}
