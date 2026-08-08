package smtp

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"log/slog"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0funct0ry/maelsink/internal/store"
)

// testClient drives a session over an in-memory net.Pipe(), talking real
// SMTP text without a real TCP socket (used for fast protocol-level tests;
// server_test.go covers the real-socket integration path).
type testClient struct {
	t    *testing.T
	conn net.Conn
	r    *bufio.Reader
}

func newTestSession(t *testing.T, cfg Config) (*testClient, store.MessageStore) {
	t.Helper()
	st := store.NewMemoryStore()
	srv := &Server{cfg: cfg, store: st, publisher: store.NoopPublisher{}, logger: slog.New(slog.DiscardHandler)}

	clientConn, serverConn := net.Pipe()
	sess := newSession(srv, serverConn)
	go sess.run()

	return &testClient{t: t, conn: clientConn, r: bufio.NewReader(clientConn)}, st
}

func (c *testClient) send(line string) {
	c.t.Helper()
	if _, err := c.conn.Write([]byte(line + "\r\n")); err != nil {
		c.t.Fatalf("write %q: %v", line, err)
	}
}

// readReply reads one SMTP reply, following multi-line "###-" continuations.
func (c *testClient) readReply() (code int, lines []string) {
	c.t.Helper()
	for {
		line, err := c.r.ReadString('\n')
		if err != nil {
			c.t.Fatalf("read reply: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if len(line) < 4 {
			c.t.Fatalf("malformed reply line %q", line)
		}
		n, err := strconv.Atoi(line[:3])
		if err != nil {
			c.t.Fatalf("malformed reply code in %q: %v", line, err)
		}
		code = n
		lines = append(lines, line[4:])
		if line[3] == ' ' {
			return code, lines
		}
	}
}

func (c *testClient) expectCode(want int) []string {
	c.t.Helper()
	code, lines := c.readReply()
	if code != want {
		c.t.Fatalf("got code %d %v, want %d", code, lines, want)
	}
	return lines
}

func testConfig() Config {
	return Config{
		Host:           "127.0.0.1",
		Port:           1025,
		Domain:         "maelsink.test",
		MaxMessageSize: 1024,
	}
}

func TestSession_HappyPath(t *testing.T) {
	c, st := newTestSession(t, testConfig())
	defer c.conn.Close()

	c.expectCode(codeGreeting)

	c.send("EHLO client.example.com")
	c.expectCode(codeOK)

	c.send("MAIL FROM:<alice@example.com>")
	c.expectCode(codeOK)

	c.send("RCPT TO:<bob@example.com>")
	c.expectCode(codeOK)

	c.send("DATA")
	c.expectCode(codeStartMailInput)

	c.send("Subject: hi")
	c.send("")
	c.send("hello world")
	c.send(".")
	c.expectCode(codeOK)

	c.send("QUIT")
	c.expectCode(codeClosing)

	_, total, err := st.List(context.Background(), store.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
}

func TestSession_RCPTNeverRejectsUnknownRecipient(t *testing.T) {
	c, _ := newTestSession(t, testConfig())
	defer c.conn.Close()

	c.expectCode(codeGreeting)
	c.send("HELO client")
	c.expectCode(codeOK)
	c.send("MAIL FROM:<a@example.com>")
	c.expectCode(codeOK)

	c.send("RCPT TO:<nobody@nowhere.invalid>")
	c.expectCode(codeOK)
}

func TestSession_OutOfOrderCommands(t *testing.T) {
	c, _ := newTestSession(t, testConfig())
	defer c.conn.Close()

	c.expectCode(codeGreeting)
	c.send("HELO client")
	c.expectCode(codeOK)

	c.send("RCPT TO:<bob@example.com>")
	c.expectCode(codeBadSequence)

	c.send("DATA")
	c.expectCode(codeBadSequence)
}

func TestSession_OversizedDataRejected(t *testing.T) {
	cfg := testConfig()
	cfg.MaxMessageSize = 10
	c, st := newTestSession(t, cfg)
	defer c.conn.Close()

	c.expectCode(codeGreeting)
	c.send("HELO client")
	c.expectCode(codeOK)
	c.send("MAIL FROM:<a@example.com>")
	c.expectCode(codeOK)
	c.send("RCPT TO:<b@example.com>")
	c.expectCode(codeOK)
	c.send("DATA")
	c.expectCode(codeStartMailInput)

	// No terminating "." is sent: the server detects the size overflow
	// while still consuming the first (over-limit) line and immediately
	// replies + closes the connection without waiting for the dot
	// terminator, so a real client's terminator write would land on an
	// already-closing TCP connection rather than block — unlike
	// net.Pipe's unbuffered, synchronous semantics used here.
	c.send("this line is definitely longer than ten bytes")
	c.expectCode(codeExceededStorage)

	_, total, _ := st.List(context.Background(), store.ListFilter{})
	if total != 0 {
		t.Fatalf("total = %d, want 0 (oversized message must not be saved)", total)
	}
}

func TestSession_AuthRequiredBeforeMail(t *testing.T) {
	cfg := testConfig()
	cfg.AuthEnabled = true
	cfg.AuthUsername = "user"
	cfg.AuthPassword = "pass"
	c, _ := newTestSession(t, cfg)
	defer c.conn.Close()

	c.expectCode(codeGreeting)
	c.send("HELO client")
	c.expectCode(codeOK)

	c.send("MAIL FROM:<a@example.com>")
	c.expectCode(codeAuthRequired)
}

func TestSession_AuthPlainSuccessAndFailure(t *testing.T) {
	cfg := testConfig()
	cfg.AuthEnabled = true
	cfg.AuthUsername = "user"
	cfg.AuthPassword = "pass"
	c, _ := newTestSession(t, cfg)
	defer c.conn.Close()

	c.expectCode(codeGreeting)
	c.send("EHLO client")
	c.expectCode(codeOK)

	badBlob := base64.StdEncoding.EncodeToString([]byte("\x00user\x00wrong"))
	c.send("AUTH PLAIN " + badBlob)
	c.expectCode(codeAuthFailed)

	goodBlob := base64.StdEncoding.EncodeToString([]byte("\x00user\x00pass"))
	c.send("AUTH PLAIN " + goodBlob)
	c.expectCode(codeAuthSuccess)

	c.send("MAIL FROM:<a@example.com>")
	c.expectCode(codeOK)
}

func TestSession_AuthLoginSuccess(t *testing.T) {
	cfg := testConfig()
	cfg.AuthEnabled = true
	cfg.AuthUsername = "user"
	cfg.AuthPassword = "pass"
	c, _ := newTestSession(t, cfg)
	defer c.conn.Close()

	c.expectCode(codeGreeting)
	c.send("EHLO client")
	c.expectCode(codeOK)

	c.send("AUTH LOGIN")
	c.expectCode(codeAuthContinue)
	c.send(base64.StdEncoding.EncodeToString([]byte("user")))
	c.expectCode(codeAuthContinue)
	c.send(base64.StdEncoding.EncodeToString([]byte("pass")))
	c.expectCode(codeAuthSuccess)
}

func TestSession_AuthNotRequiredWhenDisabled(t *testing.T) {
	c, _ := newTestSession(t, testConfig())
	defer c.conn.Close()

	c.expectCode(codeGreeting)
	c.send("HELO client")
	c.expectCode(codeOK)
	c.send("MAIL FROM:<a@example.com>")
	c.expectCode(codeOK)
}

func TestSession_VRFYAlwaysSucceeds(t *testing.T) {
	c, _ := newTestSession(t, testConfig())
	defer c.conn.Close()

	c.expectCode(codeGreeting)
	c.send("VRFY nonexistent-user")
	c.expectCode(codeCannotVRFY)
}

func TestSession_NoopAndRset(t *testing.T) {
	c, _ := newTestSession(t, testConfig())
	defer c.conn.Close()

	c.expectCode(codeGreeting)
	c.send("NOOP")
	c.expectCode(codeOK)
	c.send("RSET")
	c.expectCode(codeOK)
}

func TestSession_MalformedMessageStillStored(t *testing.T) {
	c, st := newTestSession(t, testConfig())
	defer c.conn.Close()

	c.expectCode(codeGreeting)
	c.send("HELO client")
	c.expectCode(codeOK)
	c.send("MAIL FROM:<a@example.com>")
	c.expectCode(codeOK)
	c.send("RCPT TO:<b@example.com>")
	c.expectCode(codeOK)
	c.send("DATA")
	c.expectCode(codeStartMailInput)
	c.send("Content-Type: multipart/mixed; boundary=broken")
	c.send("")
	c.send("this does not respect the boundary at all")
	c.send(".")
	c.expectCode(codeOK)

	msgs, total, err := st.List(context.Background(), store.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if !msgs[0].ParseWarning {
		t.Error("expected ParseWarning=true on malformed message")
	}
}

func TestSession_StartTLS(t *testing.T) {
	certPath, keyPath := writeTestCert(t)
	cfg := testConfig()
	cfg.StartTLS = true
	cfg.TLSCert = certPath
	cfg.TLSKey = keyPath

	st := store.NewMemoryStore()
	srv := &Server{cfg: cfg, store: st, publisher: store.NoopPublisher{}, logger: slog.New(slog.DiscardHandler)}

	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() { clientConn.Close() })
	sess := newSession(srv, serverConn)
	go sess.run()

	c := &testClient{t: t, conn: clientConn, r: bufio.NewReader(clientConn)}
	c.expectCode(codeGreeting)
	c.send("EHLO client")
	lines := c.expectCode(codeOK)
	found := false
	for _, l := range lines {
		if strings.Contains(l, "STARTTLS") {
			found = true
		}
	}
	if !found {
		t.Fatalf("EHLO response missing STARTTLS capability: %v", lines)
	}

	c.send("STARTTLS")
	c.expectCode(codeGreeting)

	tlsClient := tls.Client(clientConn, &tls.Config{InsecureSkipVerify: true})
	if err := tlsClient.Handshake(); err != nil {
		t.Fatalf("client TLS handshake: %v", err)
	}
	c.conn = tlsClient
	c.r = bufio.NewReader(tlsClient)

	c.send("EHLO client")
	c.expectCode(codeOK)
	c.send("MAIL FROM:<a@example.com>")
	c.expectCode(codeOK)
}

// writeTestCert generates a throwaway self-signed cert/key pair for
// STARTTLS tests.
func writeTestCert(t *testing.T) (certPath, keyPath string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big1,
		Subject:      pkix.Name{CommonName: "localhost"},
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

var big1 = mustBigInt(1)

func mustBigInt(n int64) *big.Int {
	return big.NewInt(n)
}
