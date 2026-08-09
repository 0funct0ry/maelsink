package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestShell_ExecuteAgainstRunningServer(t *testing.T) {
	fixedTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	srv, _ := newTestAPIServer(t,
		fixtureMessage("msg_1", "alice@example.com", "bob@example.com", "Hello there", fixedTime),
	)

	stdout, stderr, err := execCommand(t, "", "shell",
		"--api", srv.URL,
		"-e", "list --limit 5",
		"-e", "stats",
	)
	if err != nil {
		t.Fatalf("execCommand: err=%v stdout=%s stderr=%s", err, stdout, stderr)
	}
}

func TestShell_ExecuteAgainstUnreachableServer(t *testing.T) {
	stdout, stderr, err := execCommand(t, "", "shell",
		"--api", "http://127.0.0.1:1",
		"-e", "list --limit 5",
		"-e", "stats",
	)
	if err == nil {
		t.Fatalf("expected non-zero exit, got success; stdout=%s stderr=%s", stdout, stderr)
	}
	if strings.Contains(stderr, "message_not_found") || strings.Contains(stderr, "404") {
		t.Fatalf("expected an unreachable-style message, got an HTTP-error-style message instead:\n%s", stderr)
	}
	if !strings.Contains(stderr, "could not reach") && !strings.Contains(stderr, "unreachable") {
		t.Fatalf("expected stderr to clearly indicate the server is unreachable, got:\n%s", stderr)
	}
}
