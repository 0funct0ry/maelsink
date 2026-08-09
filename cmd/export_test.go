package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExport_DefaultPath(t *testing.T) {
	srv, _ := newTestAPIServer(t, fixtureMessage("msg_1", "a@b.com", "c@d.com", "Hi", time.Now()))

	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(cwd) })

	stdout, stderr, err := execCommand(t, "", "export", "msg_1", "--api", srv.URL)
	if err != nil {
		t.Fatalf("execCommand: err=%v stderr=%s", err, stderr)
	}
	wantPath := "./msg_1.eml"
	if strings.TrimSpace(stdout) != wantPath {
		t.Fatalf("stdout = %q, want %q", strings.TrimSpace(stdout), wantPath)
	}
	data, err := os.ReadFile(filepath.Join(dir, "msg_1.eml"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "Subject: Hi") {
		t.Fatalf("unexpected .eml content: %q", data)
	}
}

func TestExport_CustomOutput(t *testing.T) {
	srv, _ := newTestAPIServer(t, fixtureMessage("msg_1", "a@b.com", "c@d.com", "Hi", time.Now()))

	dir := t.TempDir()
	out := filepath.Join(dir, "custom.eml")

	stdout, stderr, err := execCommand(t, "", "export", "msg_1", "--api", srv.URL, "-o", out)
	if err != nil {
		t.Fatalf("execCommand: err=%v stderr=%s", err, stderr)
	}
	if strings.TrimSpace(stdout) != out {
		t.Fatalf("stdout = %q, want %q", strings.TrimSpace(stdout), out)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected file at %s: %v", out, err)
	}
}

func TestExport_NotFound(t *testing.T) {
	srv, _ := newTestAPIServer(t)

	_, stderr, err := execCommand(t, "", "export", "bogus-id", "--api", srv.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "404") {
		t.Fatalf("expected 404, got stderr:\n%s", stderr)
	}
}
