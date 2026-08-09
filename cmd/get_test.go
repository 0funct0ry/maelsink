package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestGet_Table(t *testing.T) {
	srv, _ := newTestAPIServer(t, fixtureMessage("msg_1", "a@b.com", "c@d.com", "Hi there", time.Now()))

	stdout, stderr, err := execCommand(t, "", "get", "msg_1", "--api", srv.URL)
	if err != nil {
		t.Fatalf("execCommand: err=%v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "msg_1") || !strings.Contains(stdout, "Hi there") {
		t.Fatalf("missing detail fields, got:\n%s", stdout)
	}
}

func TestGet_JSON(t *testing.T) {
	srv, _ := newTestAPIServer(t, fixtureMessage("msg_1", "a@b.com", "c@d.com", "Hi", time.Now()))

	stdout, stderr, err := execCommand(t, "", "get", "msg_1", "--api", srv.URL, "--format", "json")
	if err != nil {
		t.Fatalf("execCommand: err=%v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, `"id": "msg_1"`) {
		t.Fatalf("expected JSON output, got:\n%s", stdout)
	}
}

func TestGet_ByIDPrefix(t *testing.T) {
	srv, _ := newTestAPIServer(t, fixtureMessage("14e3674db144f85bfef4d788", "a@b.com", "c@d.com", "Hi there", time.Now()))

	stdout, stderr, err := execCommand(t, "", "get", "14e3674d", "--api", srv.URL)
	if err != nil {
		t.Fatalf("execCommand: err=%v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "14e3674db144f85bfef4d788") {
		t.Fatalf("expected full id resolved from prefix, got:\n%s", stdout)
	}
}

func TestGet_AmbiguousIDPrefix(t *testing.T) {
	srv, _ := newTestAPIServer(t,
		fixtureMessage("aaaa1111aaaa1111aaaa1111", "a@b.com", "c@d.com", "One", time.Now()),
		fixtureMessage("aaaa2222aaaa2222aaaa2222", "a@b.com", "c@d.com", "Two", time.Now()),
	)

	_, stderr, err := execCommand(t, "", "get", "aaaa", "--api", srv.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "ambiguous_id") {
		t.Fatalf("expected ambiguous_id error, got stderr:\n%s", stderr)
	}
}

func TestGet_NotFound(t *testing.T) {
	srv, _ := newTestAPIServer(t)

	_, stderr, err := execCommand(t, "", "get", "bogus-id", "--api", srv.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "404") || !strings.Contains(stderr, "message_not_found") {
		t.Fatalf("expected 404 message_not_found, got stderr:\n%s", stderr)
	}
}

func TestGet_ServerUnreachable(t *testing.T) {
	_, stderr, err := execCommand(t, "", "get", "msg_1", "--api", "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "could not reach maelsink API") {
		t.Fatalf("expected unreachable message, got stderr:\n%s", stderr)
	}
}
