package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHistoryAdjacentDedupe(t *testing.T) {
	h := &History{max: 100}
	h.Add("list")
	h.Add("list")
	h.Add("show 1")
	h.Add("list")

	got := h.Lines()
	want := []string{"list", "show 1", "list"}
	if !equalSlices(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRedact(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{name: "plain command", line: "list --limit 5", want: true},
		{name: "auth-user separate token", line: "send --auth-user foo", want: false},
		{name: "api-key equals form", line: "send --api-key=secret", want: false},
		{name: "auth-pass separate token", line: "send --auth-pass hunter2", want: false},
		{name: "unrelated flag containing substring not at boundary", line: "list --auth-username-thing", want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Redact(tc.line); got != tc.want {
				t.Errorf("Redact(%q) = %v, want %v", tc.line, got, tc.want)
			}
		})
	}
}

func TestHistorySaveTrimAndMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hist")

	h := &History{path: path, max: 2}
	h.Add("one")
	h.Add("two")
	h.Add("three")

	if err := h.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	want := []string{"two", "three"}
	if !equalSlices(lines, want) {
		t.Errorf("got %v, want %v (oldest-first trim to max)", lines, want)
	}
}

func TestHistoryRedactsSecretFromDiskButNotMemory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hist")

	h := &History{path: path, max: 100}
	h.Add("list --limit 5")
	h.Add("send --api-key=topsecret --to a@b.com")

	if err := h.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(data), "topsecret") {
		t.Errorf("persisted history file contains secret: %s", data)
	}

	found := false
	for _, l := range h.Lines() {
		if strings.Contains(l, "topsecret") {
			found = true
		}
	}
	if !found {
		t.Error("in-memory session history should still contain the redacted line")
	}
}

func TestLoadHistoryMissingFileIsNotError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist")
	h, err := LoadHistory(path, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(h.Lines()) != 0 {
		t.Errorf("expected empty history, got %v", h.Lines())
	}
}
