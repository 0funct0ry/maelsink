package smtp

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0funct0ry/maelsink/internal/events"
	"github.com/0funct0ry/maelsink/internal/store"
	"github.com/0funct0ry/maelsink/internal/webauth"
)

func authPlainBlob(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte("\x00" + username + "\x00" + password))
}

// TestSession_AuthPlaintextRefusedWithoutInsecureOrTLS verifies RFC 4954's
// closing MUST: AUTH PLAIN/LOGIN over an unprotected connection is refused
// with 538 unless allow_insecure or implicit TLS applies.
func TestSession_AuthPlaintextRefusedWithoutInsecureOrTLS(t *testing.T) {
	cfg := testConfig()
	cfg.AuthAllowInsecure = false
	cfg.AuthEnabled = true
	cfg.AuthUsername = "user"
	cfg.AuthPassword = "pass"
	c, _ := newTestSession(t, cfg)
	defer c.conn.Close()

	c.expectCode(codeGreeting)
	c.send("EHLO client")
	c.expectCode(codeOK)

	c.send("AUTH PLAIN " + authPlainBlob("user", "pass"))
	c.expectCode(codeEncryptionRequired)
}

// TestSession_AuthPlaintextAllowedWithAllowInsecure verifies
// smtp.auth.allow_insecure permits plaintext AUTH for local/CI use.
func TestSession_AuthPlaintextAllowedWithAllowInsecure(t *testing.T) {
	cfg := testConfig()
	cfg.AuthAllowInsecure = true
	cfg.AuthEnabled = true
	cfg.AuthUsername = "user"
	cfg.AuthPassword = "pass"
	c, _ := newTestSession(t, cfg)
	defer c.conn.Close()

	c.expectCode(codeGreeting)
	c.send("EHLO client")
	c.expectCode(codeOK)

	c.send("AUTH PLAIN " + authPlainBlob("user", "pass"))
	c.expectCode(codeAuthSuccess)
}

// TestSession_RequireStartTLSBlocksMailAndAuth verifies smtp.require_starttls
// rejects MAIL FROM and AUTH until STARTTLS has completed.
func TestSession_RequireStartTLSBlocksMailAndAuth(t *testing.T) {
	certPath, keyPath := writeTestCert(t)
	cfg := testConfig()
	cfg.TLSCert = certPath
	cfg.TLSKey = keyPath
	cfg.RequireStartTLS = true
	cfg.AuthEnabled = true
	cfg.AuthUsername = "user"
	cfg.AuthPassword = "pass"

	c, _ := newTestSession(t, cfg)
	defer c.conn.Close()

	c.expectCode(codeGreeting)
	c.send("EHLO client")
	c.expectCode(codeOK)

	c.send("MAIL FROM:<a@example.com>")
	c.expectCode(codeAuthRequired)

	c.send("AUTH PLAIN " + authPlainBlob("user", "pass"))
	c.expectCode(codeAuthRequired)

	c.send("STARTTLS")
	c.expectCode(codeGreeting)

	tlsClient := tls.Client(c.conn, &tls.Config{InsecureSkipVerify: true})
	if err := tlsClient.Handshake(); err != nil {
		t.Fatalf("client TLS handshake: %v", err)
	}
	c.conn = tlsClient
	c.r = bufio.NewReader(tlsClient)

	c.send("EHLO client")
	c.expectCode(codeOK)
	c.send("AUTH PLAIN " + authPlainBlob("user", "pass"))
	c.expectCode(codeAuthSuccess)
	c.send("MAIL FROM:<a@example.com>")
	c.expectCode(codeOK)
}

// TestServer_RequireTLSSpeaksImplicitTLSAndHidesSTARTTLS is a full
// integration test (real TCP socket) verifying smtp.require_tls wraps every
// connection in TLS from the first byte and never advertises or accepts
// STARTTLS.
func TestServer_RequireTLSSpeaksImplicitTLSAndHidesSTARTTLS(t *testing.T) {
	certPath, keyPath := writeTestCert(t)
	cfg := Config{
		Host:           "127.0.0.1",
		Port:           0,
		Domain:         "maelsink.test",
		MaxMessageSize: 1024 * 1024,
		RequireTLS:     true,
		TLSCert:        certPath,
		TLSKey:         keyPath,
	}
	st := store.NewMemoryStore()
	srv, err := New(cfg, st, events.NewBus(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe(ctx) }()
	t.Cleanup(func() { srv.Close() })

	waitForAddr(t, srv)

	rawConn, err := net.Dial("tcp", srv.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer rawConn.Close()

	tlsConn := tls.Client(rawConn, &tls.Config{InsecureSkipVerify: true})
	if err := tlsConn.Handshake(); err != nil {
		t.Fatalf("expected implicit TLS handshake to succeed: %v", err)
	}

	c := &testClient{t: t, conn: tlsConn, r: bufio.NewReader(tlsConn)}
	c.expectCode(codeGreeting)
	c.send("EHLO client")
	lines := c.expectCode(codeOK)
	for _, l := range lines {
		if strings.Contains(l, "STARTTLS") {
			t.Fatalf("EHLO response must not advertise STARTTLS in require_tls mode: %v", lines)
		}
	}

	c.send("STARTTLS")
	c.expectCode(codeBadSequence)
}

// TestSession_AcceptAnySucceedsForArbitraryCredentials verifies
// smtp.auth.accept_any succeeds regardless of the presented username or
// password, including empty credentials.
func TestSession_AcceptAnySucceedsForArbitraryCredentials(t *testing.T) {
	cfg := testConfig()
	cfg.AuthEnabled = true
	cfg.AuthAcceptAny = true
	c, _ := newTestSession(t, cfg)
	defer c.conn.Close()

	c.expectCode(codeGreeting)
	c.send("EHLO client")
	c.expectCode(codeOK)

	c.send("AUTH PLAIN " + authPlainBlob("whoever", ""))
	c.expectCode(codeAuthSuccess)
}

// TestSession_AuthFileAndExtraCredentialsAlongsideSinglePair verifies a user
// present only in smtp.auth.file and a user present only via the
// MAELSINK_SMTP_AUTH-derived extra-credentials map both authenticate
// successfully alongside the single configured username/password.
func TestSession_AuthFileAndExtraCredentialsAlongsideSinglePair(t *testing.T) {
	dir := t.TempDir()
	authFile := filepath.Join(dir, "smtp-auth")
	if err := webauth.Upsert(authFile, "filedup", "filepass"); err != nil {
		t.Fatalf("webauth.Upsert: %v", err)
	}
	if _, err := os.Stat(authFile); err != nil {
		t.Fatalf("auth file not created: %v", err)
	}

	cfg := testConfig()
	cfg.AuthEnabled = true
	cfg.AuthUsername = "single"
	cfg.AuthPassword = "singlepass"
	cfg.AuthFile = authFile
	cfg.AuthExtraCredentials = map[string]string{"envuser": "envpass"}

	for _, tc := range []struct {
		name, user, pass string
	}{
		{"single pair", "single", "singlepass"},
		{"auth file", "filedup", "filepass"},
		{"extra credentials map", "envuser", "envpass"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newTestSession(t, cfg)
			defer c.conn.Close()

			c.expectCode(codeGreeting)
			c.send("EHLO client")
			c.expectCode(codeOK)
			c.send("AUTH PLAIN " + authPlainBlob(tc.user, tc.pass))
			c.expectCode(codeAuthSuccess)
		})
	}
}

// waitForAddr blocks until srv's listener is bound, so tests dialing srv.Addr()
// don't race ListenAndServe's goroutine.
func waitForAddr(t *testing.T, srv *Server) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if srv.Addr() != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("server never bound a listener")
}
