package builtin

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0funct0ry/maelsink/internal/shell"
)

func TestHistory_Basic(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	h, err := shell.LoadHistory("", 100)
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	h.Add("list")
	h.Add("show msg_1")
	s.SetHistory(h)

	if err := (History{}).Run(context.Background(), s, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "show msg_1") {
		t.Errorf("out = %q", out.String())
	}
}

func TestHistory_Unavailable(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	if err := (History{}).Run(context.Background(), s, nil); err == nil {
		t.Fatal("expected error when no history is attached")
	}
}

func TestHistory_EditEntryInteractiveLoadsPendingBuffer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a unix shell script as a fake editor")
	}
	dir := t.TempDir()
	fakeEditor := filepath.Join(dir, "fake-editor.sh")
	if err := os.WriteFile(fakeEditor, []byte("#!/bin/sh\nsed -i.bak 's/limit 5/limit 50/' \"$1\"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	s, out, _ := newTestSession(t, nil)
	s.Cfg.Editor = fakeEditor
	s.Interactive = true

	h, err := shell.LoadHistory("", 100)
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	h.Add("list --limit 5")
	h.Add("stats")
	s.SetHistory(h)

	// Entry 1 is "list --limit 5" (1-based, matching plain "history"'s own
	// numbering).
	if err := (History{}).Run(context.Background(), s, []string{"-e", "1"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.String() != "" {
		t.Errorf("expected nothing written to Out in interactive mode, got %q", out.String())
	}
	if !strings.Contains(s.PendingBuffer, "list --limit 50") {
		t.Errorf("PendingBuffer = %q", s.PendingBuffer)
	}
}

func TestHistory_EditEntryOutOfRange(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	h, err := shell.LoadHistory("", 100)
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	h.Add("list")
	s.SetHistory(h)

	if err := (History{}).Run(context.Background(), s, []string{"-e", "5"}); err == nil {
		t.Fatal("expected error for out-of-range -e")
	}
}

func TestEdit_FileEditsInPlace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a unix shell script as a fake editor")
	}
	dir := t.TempDir()
	fakeEditor := filepath.Join(dir, "fake-editor.sh")
	if err := os.WriteFile(fakeEditor, []byte("#!/bin/sh\necho appended >> \"$1\"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	inputFile := filepath.Join(dir, "in.txt")
	if err := os.WriteFile(inputFile, []byte("original\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Run in an "interactive" session to prove -f bypasses PendingBuffer
	// entirely (unlike bare "edit") — the file itself is the destination.
	s, out, _ := newTestSession(t, nil)
	s.Cfg.Editor = fakeEditor
	s.Interactive = true

	if err := (Edit{}).Run(context.Background(), s, []string{"-f", inputFile}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	saved, err := os.ReadFile(inputFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(saved), "original") || !strings.Contains(string(saved), "appended") {
		t.Errorf("file contents = %q", saved)
	}
	if s.PendingBuffer != "" {
		t.Errorf("PendingBuffer should be untouched by -f, got %q", s.PendingBuffer)
	}
	if !strings.Contains(out.String(), inputFile) {
		t.Errorf("expected a confirmation mentioning %q, got %q", inputFile, out.String())
	}
}

func TestEdit_BareLoadsPendingBufferInsteadOfPrinting(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a unix shell script as a fake editor")
	}
	dir := t.TempDir()
	fakeEditor := filepath.Join(dir, "fake-editor.sh")
	if err := os.WriteFile(fakeEditor, []byte("#!/bin/sh\necho typed >> \"$1\"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	s, out, _ := newTestSession(t, nil)
	s.Cfg.Editor = fakeEditor
	s.Interactive = true

	if err := (Edit{}).Run(context.Background(), s, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.String() != "" {
		t.Errorf("expected nothing written to Out in interactive mode, got %q", out.String())
	}
	if !strings.Contains(s.PendingBuffer, "typed") {
		t.Errorf("PendingBuffer = %q", s.PendingBuffer)
	}
	if strings.HasSuffix(s.PendingBuffer, "\n") {
		t.Errorf("PendingBuffer must not have a trailing newline (would embed as a phantom second prompt line): %q", s.PendingBuffer)
	}
}

func TestEdit_BareBatchModePrints(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a unix shell script as a fake editor")
	}
	dir := t.TempDir()
	fakeEditor := filepath.Join(dir, "fake-editor.sh")
	if err := os.WriteFile(fakeEditor, []byte("#!/bin/sh\necho typed >> \"$1\"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	s, out, _ := newTestSession(t, nil)
	s.Cfg.Editor = fakeEditor
	s.Interactive = false

	if err := (Edit{}).Run(context.Background(), s, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "typed") {
		t.Errorf("out = %q", out.String())
	}
}

func TestSh_DisabledByDefault(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	s.Cfg.ShEnabled = false
	if err := (Sh{}).Run(context.Background(), s, []string{"echo", "hi"}); err == nil {
		t.Fatal("expected error when shell.sh_enabled is false")
	}
}

func TestSh_RunsWhenEnabled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses echo via a unix shell")
	}
	s, out, _ := newTestSession(t, nil)
	s.Cfg.ShEnabled = true
	if err := (Sh{}).Run(context.Background(), s, []string{"echo", "hi-there"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "hi-there") {
		t.Errorf("out = %q", out.String())
	}
}
