package cmd

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0funct0ry/maelsink/internal/smtp"
	"github.com/0funct0ry/maelsink/internal/store"
)

func newTestSMTPServer(t *testing.T) (host string, port int, st store.MessageStore) {
	t.Helper()
	st = store.NewMemoryStore()
	srv, err := smtp.New(smtp.Config{
		Host: "127.0.0.1", Port: 0, Domain: "maelsink.test", MaxMessageSize: 1 << 20,
	}, st, store.NoopPublisher{}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("smtp.New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.ListenAndServe(ctx) }()
	for i := 0; i < 100 && srv.Addr() == nil; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if srv.Addr() == nil {
		t.Fatal("smtp server did not start")
	}
	t.Cleanup(func() {
		cancel()
		_ = srv.Close()
		<-done
	})

	_, portStr, _ := splitHostPort(srv.Addr().String())
	port, _ = strconv.Atoi(portStr)
	return "127.0.0.1", port, st
}

func splitHostPort(addr string) (host, port string, err error) {
	i := strings.LastIndex(addr, ":")
	return addr[:i], addr[i+1:], nil
}

func TestSend_Flags(t *testing.T) {
	host, port, st := newTestSMTPServer(t)

	_, stderr, err := execCommand(t, "", "send",
		"--smtp-host", host, "--smtp-port", strconv.Itoa(port),
		"--from", "sender@example.com", "--to", "rcpt@example.com",
		"--subject", "flag test", "--text", "hello from flags")
	if err != nil {
		t.Fatalf("execCommand: err=%v stderr=%s", err, stderr)
	}

	_, total, err := st.List(context.Background(), store.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
}

func TestSend_Attachment(t *testing.T) {
	host, port, st := newTestSMTPServer(t)

	dir := t.TempDir()
	attPath := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(attPath, []byte("attachment body"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := execCommand(t, "", "send",
		"--smtp-host", host, "--smtp-port", strconv.Itoa(port),
		"--from", "sender@example.com", "--to", "rcpt@example.com",
		"--subject", "with attach", "--text", "body", "--attach", attPath)
	if err != nil {
		t.Fatalf("execCommand: err=%v stderr=%s", err, stderr)
	}

	msgs, _, err := st.List(context.Background(), store.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(msgs) != 1 || len(msgs[0].Attachments) != 1 {
		t.Fatalf("expected 1 message with 1 attachment, got %+v", msgs)
	}
	if msgs[0].Attachments[0].Filename != "note.txt" {
		t.Fatalf("attachment filename = %q", msgs[0].Attachments[0].Filename)
	}
}

func TestSend_RawStdin(t *testing.T) {
	host, port, st := newTestSMTPServer(t)

	raw := "From: sender@example.com\r\nTo: rcpt@example.com\r\nSubject: raw test\r\n\r\nraw body\r\n"

	_, stderr, err := execCommand(t, raw, "send",
		"--smtp-host", host, "--smtp-port", strconv.Itoa(port), "--raw")
	if err != nil {
		t.Fatalf("execCommand: err=%v stderr=%s", err, stderr)
	}

	msgs, total, err := st.List(context.Background(), store.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 || !strings.Contains(msgs[0].TextBody, "raw body") {
		t.Fatalf("unexpected result: total=%d msgs=%+v", total, msgs)
	}
}

func TestSend_JSONFile(t *testing.T) {
	host, port, st := newTestSMTPServer(t)

	dir := t.TempDir()
	specPath := filepath.Join(dir, "message.json")
	spec := map[string]any{
		"from":    "sender@example.com",
		"to":      []string{"rcpt@example.com"},
		"subject": "json file test",
		"text":    "body from json",
	}
	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(specPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := execCommand(t, "", "send",
		"--smtp-host", host, "--smtp-port", strconv.Itoa(port), "--file", specPath)
	if err != nil {
		t.Fatalf("execCommand: err=%v stderr=%s", err, stderr)
	}

	msgs, total, err := st.List(context.Background(), store.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 || !strings.Contains(msgs[0].TextBody, "body from json") {
		t.Fatalf("unexpected result: total=%d msgs=%+v", total, msgs)
	}
}

func TestSend_RejectedByServer(t *testing.T) {
	host, port, _ := newTestSMTPServer(t)

	// Oversized message triggers a 552 rejection from the real SMTP server.
	big := strings.Repeat("a", 2<<20)
	_, stderr, err := execCommand(t, "", "send",
		"--smtp-host", host, "--smtp-port", strconv.Itoa(port),
		"--from", "sender@example.com", "--to", "rcpt@example.com",
		"--subject", "too big", "--text", big)
	if err == nil {
		t.Fatal("expected error for oversized message")
	}
	if !strings.Contains(stderr, "send failed") {
		t.Fatalf("expected send failure message, got stderr:\n%s", stderr)
	}
}
