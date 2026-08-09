package cmd

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/0funct0ry/maelsink/internal/store"
)

func TestDelete_Success(t *testing.T) {
	srv, st := newTestAPIServer(t, fixtureMessage("msg_1", "a@b.com", "c@d.com", "Hi", time.Now()))

	stdout, stderr, err := execCommand(t, "", "delete", "msg_1", "--api", srv.URL)
	if err != nil {
		t.Fatalf("execCommand: err=%v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "deleted msg_1") {
		t.Fatalf("expected confirmation, got:\n%s", stdout)
	}
	if _, err := st.Get(context.Background(), "msg_1"); err != store.ErrNotFound {
		t.Fatalf("expected message to be deleted, Get err = %v", err)
	}
}

func TestDelete_ByIDPrefix(t *testing.T) {
	srv, st := newTestAPIServer(t, fixtureMessage("14e3674db144f85bfef4d788", "a@b.com", "c@d.com", "Hi", time.Now()))

	_, stderr, err := execCommand(t, "", "delete", "14e3674d", "--api", srv.URL)
	if err != nil {
		t.Fatalf("execCommand: err=%v stderr=%s", err, stderr)
	}
	if _, err := st.Get(context.Background(), "14e3674db144f85bfef4d788"); err != store.ErrNotFound {
		t.Fatalf("expected message deleted via prefix, Get err = %v", err)
	}
}

func TestDelete_AmbiguousIDPrefix(t *testing.T) {
	srv, st := newTestAPIServer(t,
		fixtureMessage("aaaa1111aaaa1111aaaa1111", "a@b.com", "c@d.com", "One", time.Now()),
		fixtureMessage("aaaa2222aaaa2222aaaa2222", "a@b.com", "c@d.com", "Two", time.Now()),
	)

	_, stderr, err := execCommand(t, "", "delete", "aaaa", "--api", srv.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "ambiguous_id") {
		t.Fatalf("expected ambiguous_id error, got stderr:\n%s", stderr)
	}
	if _, err := st.Get(context.Background(), "aaaa1111aaaa1111aaaa1111"); err != nil {
		t.Fatalf("expected message to survive failed ambiguous delete: %v", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	srv, _ := newTestAPIServer(t)

	_, stderr, err := execCommand(t, "", "delete", "bogus-id", "--api", srv.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr, "404") {
		t.Fatalf("expected 404, got stderr:\n%s", stderr)
	}
}
