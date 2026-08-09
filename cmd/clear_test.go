package cmd

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestClear_Yes(t *testing.T) {
	srv, st := newTestAPIServer(t,
		fixtureMessage("msg_1", "a@b.com", "c@d.com", "Hi", time.Now()),
		fixtureMessage("msg_2", "a@b.com", "c@d.com", "Hi 2", time.Now()),
	)

	stdout, stderr, err := execCommand(t, "", "clear", "--api", srv.URL, "--yes")
	if err != nil {
		t.Fatalf("execCommand: err=%v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "All messages deleted.") {
		t.Fatalf("expected confirmation, got:\n%s", stdout)
	}
	stats, err := st.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.TotalMessages != 0 {
		t.Fatalf("expected all messages cleared, TotalMessages = %d", stats.TotalMessages)
	}
}

func TestClear_InteractiveConfirm(t *testing.T) {
	srv, _ := newTestAPIServer(t, fixtureMessage("msg_1", "a@b.com", "c@d.com", "Hi", time.Now()))

	stdout, stderr, err := execCommand(t, "y\n", "clear", "--api", srv.URL)
	if err != nil {
		t.Fatalf("execCommand: err=%v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "This will delete 1 messages") {
		t.Fatalf("expected prompt, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "All messages deleted.") {
		t.Fatalf("expected confirmation after yes, got:\n%s", stdout)
	}
}

func TestClear_InteractiveAbort(t *testing.T) {
	srv, st := newTestAPIServer(t, fixtureMessage("msg_1", "a@b.com", "c@d.com", "Hi", time.Now()))

	stdout, stderr, err := execCommand(t, "n\n", "clear", "--api", srv.URL)
	if err != nil {
		t.Fatalf("execCommand: err=%v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Aborted.") {
		t.Fatalf("expected abort message, got:\n%s", stdout)
	}

	stats, err := st.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.TotalMessages != 1 {
		t.Fatalf("expected message to survive abort, TotalMessages = %d", stats.TotalMessages)
	}
}
