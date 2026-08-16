package cmd

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0funct0ry/maelsink/internal/events"
	"github.com/0funct0ry/maelsink/internal/smtp"
	"github.com/0funct0ry/maelsink/internal/store"
)

// writeTestCert generates a throwaway self-signed cert/key pair, mirroring
// internal/smtp/session_test.go's helper of the same purpose, for tests that
// need a real --smtp-tls-cert/--smtp-tls-key pair on the test server.
func writeTestCert(t *testing.T) (certPath, keyPath string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	dir := t.TempDir()
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")

	certOut, err := os.Create(certPath)
	if err != nil {
		t.Fatalf("create cert file: %v", err)
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		t.Fatalf("encode cert: %v", err)
	}

	keyOut, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("create key file: %v", err)
	}
	defer keyOut.Close()
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
		t.Fatalf("encode key: %v", err)
	}
	return certPath, keyPath
}

func newTestSMTPServer(t *testing.T) (host string, port int, st store.MessageStore) {
	t.Helper()
	return newTestSMTPServerWithConfig(t, smtp.Config{})
}

// newTestSMTPServerWithConfig starts a real internal/smtp server with cfg's
// Host/Port/Domain/MaxMessageSize defaulted, letting callers layer in
// TLS/auth fields (e.g. RequireTLS, RequireStartTLS) to exercise M8.10a's CLI
// TLS support against the real M8.10-hardened server.
func newTestSMTPServerWithConfig(t *testing.T, cfg smtp.Config) (host string, port int, st store.MessageStore) {
	t.Helper()
	st = store.NewMemoryStore()
	cfg.Host = "127.0.0.1"
	cfg.Port = 0
	if cfg.Domain == "" {
		cfg.Domain = "maelsink.test"
	}
	if cfg.MaxMessageSize == 0 {
		cfg.MaxMessageSize = 1 << 20
	}
	srv, err := smtp.New(cfg, st, events.NewBus(), slog.New(slog.DiscardHandler))
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

func TestSend_SMTPTLS_Implicit(t *testing.T) {
	certPath, keyPath := writeTestCert(t)
	host, port, st := newTestSMTPServerWithConfig(t, smtp.Config{
		RequireTLS: true, TLSCert: certPath, TLSKey: keyPath,
	})

	_, stderr, err := execCommand(t, "", "send",
		"--smtp-host", host, "--smtp-port", strconv.Itoa(port),
		"--smtp-tls", "implicit", "--smtp-tls-insecure-skip-verify",
		"--from", "sender@example.com", "--to", "rcpt@example.com",
		"--subject", "implicit tls", "--text", "hello over implicit tls")
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

func TestSend_SMTPTLS_StartTLS(t *testing.T) {
	certPath, keyPath := writeTestCert(t)
	host, port, st := newTestSMTPServerWithConfig(t, smtp.Config{
		RequireStartTLS: true, TLSCert: certPath, TLSKey: keyPath,
	})

	_, stderr, err := execCommand(t, "", "send",
		"--smtp-host", host, "--smtp-port", strconv.Itoa(port),
		"--smtp-tls", "starttls", "--smtp-tls-insecure-skip-verify",
		"--from", "sender@example.com", "--to", "rcpt@example.com",
		"--subject", "starttls", "--text", "hello over starttls")
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

func TestSend_SMTPTLS_RequireStartTLS_FailsWithoutFlag(t *testing.T) {
	certPath, keyPath := writeTestCert(t)
	host, port, _ := newTestSMTPServerWithConfig(t, smtp.Config{
		RequireStartTLS: true, TLSCert: certPath, TLSKey: keyPath,
	})

	_, stderr, err := execCommand(t, "", "send",
		"--smtp-host", host, "--smtp-port", strconv.Itoa(port),
		"--from", "sender@example.com", "--to", "rcpt@example.com",
		"--subject", "no tls", "--text", "should be rejected")
	if err == nil {
		t.Fatal("expected an error when --smtp-tls is omitted against a require-starttls server")
	}
	if !strings.Contains(stderr, "send failed") {
		t.Fatalf("expected a clear send failure message, got stderr:\n%s", stderr)
	}
}

func TestSend_InvalidSMTPTLSValue(t *testing.T) {
	host, port, _ := newTestSMTPServer(t)

	_, _, err := execCommand(t, "", "send",
		"--smtp-host", host, "--smtp-port", strconv.Itoa(port),
		"--smtp-tls", "bogus",
		"--from", "sender@example.com", "--to", "rcpt@example.com",
		"--subject", "x", "--text", "x")
	if err == nil {
		t.Fatal("expected an error for an invalid --smtp-tls value")
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
