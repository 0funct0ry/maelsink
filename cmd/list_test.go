package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestList_Table(t *testing.T) {
	fixedTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	srv, _ := newTestAPIServer(t,
		fixtureMessage("msg_1", "alice@example.com", "bob@example.com", "Hello there", fixedTime),
		fixtureMessage("msg_2", "carol@example.com", "dave@example.com", "Second message", fixedTime.Add(time.Minute)),
	)

	stdout, stderr, err := execCommand(t, "", "list", "--api", srv.URL)
	if err != nil {
		t.Fatalf("execCommand: err=%v stderr=%s", err, stderr)
	}

	if !strings.Contains(stdout, "ID") || !strings.Contains(stdout, "FROM") {
		t.Fatalf("missing table header, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "msg_1") || !strings.Contains(stdout, "msg_2") {
		t.Fatalf("missing message rows, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "alice@example.com") || !strings.Contains(stdout, "carol@example.com") {
		t.Fatalf("missing from addresses, got:\n%s", stdout)
	}
}

func TestList_JSON(t *testing.T) {
	fixedTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	srv, _ := newTestAPIServer(t, fixtureMessage("msg_1", "a@b.com", "c@d.com", "Hi", fixedTime))

	stdout, stderr, err := execCommand(t, "", "list", "--api", srv.URL, "--format", "json")
	if err != nil {
		t.Fatalf("execCommand: err=%v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, `"messages"`) || !strings.Contains(stdout, `"total"`) {
		t.Fatalf("expected wrapped envelope, got:\n%s", stdout)
	}
}

func TestList_Template(t *testing.T) {
	fixedTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	srv, _ := newTestAPIServer(t,
		fixtureMessage("msg_1", "alice@example.com", "bob@example.com", "Hello there", fixedTime),
		fixtureMessage("msg_2", "carol@example.com", "dave@example.com", "Second message", fixedTime.Add(time.Minute)),
	)

	stdout, stderr, err := execCommand(t, "", "list", "--api", srv.URL, "--format", "{{.ID}}: {{.From}}")
	if err != nil {
		t.Fatalf("execCommand: err=%v stderr=%s", err, stderr)
	}

	want := "msg_2: carol@example.com\nmsg_1: alice@example.com\n"
	if stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
}

func TestList_Template_InvalidSyntax(t *testing.T) {
	srv, _ := newTestAPIServer(t, fixtureMessage("msg_1", "a@b.com", "c@d.com", "Hi", time.Now()))

	_, stderr, err := execCommand(t, "", "list", "--api", srv.URL, "--format", "{{.ID")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "parsing --format template") {
		t.Fatalf("expected template parse error, got stderr:\n%s", stderr)
	}
}

func TestList_EmptyState(t *testing.T) {
	srv, _ := newTestAPIServer(t)

	stdout, stderr, err := execCommand(t, "", "list", "--api", srv.URL)
	if err != nil {
		t.Fatalf("execCommand: err=%v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "No messages.") {
		t.Fatalf("expected empty state message, got:\n%s", stdout)
	}
}

func TestList_ServerUnreachable(t *testing.T) {
	_, stderr, err := execCommand(t, "", "list", "--api", "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "could not reach maelsink API") {
		t.Fatalf("expected unreachable message, got stderr:\n%s", stderr)
	}
}
