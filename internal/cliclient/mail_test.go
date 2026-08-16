package cliclient

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/mail"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMessageSpec_Build_TextOnly(t *testing.T) {
	spec := MessageSpec{From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi", Text: "hello"}
	raw, err := spec.Build(time.Now())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	m, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if got := m.Header.Get("Subject"); got != "hi" {
		t.Errorf("Subject = %q", got)
	}
	body, _ := readAll(m.Body)
	if !strings.Contains(body, "hello") {
		t.Errorf("body = %q", body)
	}
}

func TestMessageSpec_Build_Tags(t *testing.T) {
	spec := MessageSpec{From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi", Text: "hello", Tags: []string{"smoke", "release"}}
	raw, err := spec.Build(time.Now())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	m, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	got := m.Header[textproto.CanonicalMIMEHeaderKey("X-Tag")]
	if len(got) != 2 || got[0] != "smoke" || got[1] != "release" {
		t.Fatalf("X-Tag headers = %v, want [smoke release]", got)
	}
}

func TestMessageSpec_Build_TextAndHTML(t *testing.T) {
	spec := MessageSpec{From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi", Text: "plain", HTML: "<b>html</b>"}
	raw, err := spec.Build(time.Now())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	s := string(raw)
	if !strings.Contains(s, "multipart/alternative") {
		t.Errorf("expected multipart/alternative, got:\n%s", s)
	}
	if !strings.Contains(s, "plain") || !strings.Contains(s, "<b>html</b>") {
		t.Errorf("expected both bodies present, got:\n%s", s)
	}
}

func TestMessageSpec_Build_WithAttachment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invoice.txt")
	if err := os.WriteFile(path, []byte("invoice contents"), 0o644); err != nil {
		t.Fatal(err)
	}

	spec := MessageSpec{
		From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi", Text: "hello",
		Attachments: []AttachmentSpec{{Path: path}},
	}
	raw, err := spec.Build(time.Now())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	s := string(raw)
	if !strings.Contains(s, "multipart/mixed") {
		t.Errorf("expected multipart/mixed, got:\n%s", s)
	}
	if !strings.Contains(s, `filename="invoice.txt"`) {
		t.Errorf("expected filename in Content-Disposition, got:\n%s", s)
	}
}

func TestMessageSpec_Envelope(t *testing.T) {
	spec := MessageSpec{From: "a@b.com", To: []string{"x@y.com"}, Cc: []string{"z@y.com"}, Bcc: []string{"w@y.com"}}
	from, to := spec.Envelope()
	if from != "a@b.com" {
		t.Errorf("from = %q", from)
	}
	want := []string{"x@y.com", "z@y.com", "w@y.com"}
	if len(to) != len(want) {
		t.Fatalf("to = %v, want %v", to, want)
	}
	for i := range want {
		if to[i] != want[i] {
			t.Errorf("to[%d] = %q, want %q", i, to[i], want[i])
		}
	}
}

// fakeSMTPServer accepts one SMTP transaction over a raw TCP listener,
// capturing MAIL FROM / RCPT TO / DATA without a real store — enough to
// verify cliclient.Send's wire behavior in isolation from internal/smtp.
func fakeSMTPServer(t *testing.T) (addr string, dataCh <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ch := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		tp := textproto.NewConn(conn)
		tp.PrintfLine("220 fake.local ESMTP")
		for {
			line, err := tp.ReadLine()
			if err != nil {
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
				tp.PrintfLine("250 fake.local")
			case strings.HasPrefix(line, "MAIL FROM"):
				tp.PrintfLine("250 OK")
			case strings.HasPrefix(line, "RCPT TO"):
				tp.PrintfLine("250 OK")
			case line == "DATA":
				tp.PrintfLine("354 go ahead")
				dr := tp.DotReader()
				data, _ := readAllBytes(dr)
				ch <- string(data)
				tp.PrintfLine("250 OK")
			case line == "QUIT":
				tp.PrintfLine("221 bye")
				return
			}
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return ln.Addr().String(), ch
}

func TestSend_Success(t *testing.T) {
	addr, dataCh := fakeSMTPServer(t)

	spec := MessageSpec{From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi", Text: "hello"}
	raw, err := spec.Build(time.Now())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	from, to := spec.Envelope()

	if err := Send(context.Background(), addr, nil, from, to, raw); err != nil {
		t.Fatalf("Send: %v", err)
	}

	select {
	case data := <-dataCh:
		if !strings.Contains(data, "hello") {
			t.Errorf("server received unexpected data: %q", data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for DATA")
	}
}

// generateTestCert creates a throwaway self-signed cert/key pair for
// STARTTLS/implicit-TLS tests, mirroring internal/smtp/session_test.go's
// writeTestCert helper.
func generateTestCert(t *testing.T) tls.Certificate {
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
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("X509KeyPair: %v", err)
	}
	return cert
}

// fakeSMTPServerStartTLS behaves like fakeSMTPServer but advertises STARTTLS
// in its EHLO response and, on receiving the STARTTLS command, upgrades the
// connection in place before continuing the same line-based protocol loop.
func fakeSMTPServerStartTLS(t *testing.T, cert tls.Certificate) (addr string, dataCh <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ch := make(chan string, 1)
	tlsConf := &tls.Config{Certificates: []tls.Certificate{cert}}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		tp := textproto.NewConn(conn)
		tp.PrintfLine("220 fake.local ESMTP")
		for {
			line, err := tp.ReadLine()
			if err != nil {
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
				tp.PrintfLine("250-fake.local")
				tp.PrintfLine("250 STARTTLS")
			case line == "STARTTLS":
				tp.PrintfLine("220 go ahead")
				tlsConn := tls.Server(conn, tlsConf)
				if err := tlsConn.Handshake(); err != nil {
					return
				}
				tp = textproto.NewConn(tlsConn)
			case strings.HasPrefix(line, "MAIL FROM"):
				tp.PrintfLine("250 OK")
			case strings.HasPrefix(line, "RCPT TO"):
				tp.PrintfLine("250 OK")
			case line == "DATA":
				tp.PrintfLine("354 go ahead")
				dr := tp.DotReader()
				data, _ := readAllBytes(dr)
				ch <- string(data)
				tp.PrintfLine("250 OK")
			case line == "QUIT":
				tp.PrintfLine("221 bye")
				return
			}
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return ln.Addr().String(), ch
}

// fakeSMTPServerNoStartTLS is fakeSMTPServer's EHLO response without a
// STARTTLS capability line, used to verify TLSStartTLS fails clearly rather
// than silently falling back to plaintext when the server doesn't support it.
func fakeSMTPServerNoStartTLS(t *testing.T) (addr string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		tp := textproto.NewConn(conn)
		tp.PrintfLine("220 fake.local ESMTP")
		for {
			line, err := tp.ReadLine()
			if err != nil {
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
				tp.PrintfLine("250 fake.local")
			case line == "QUIT":
				tp.PrintfLine("221 bye")
				return
			default:
				// A real server rejects an unrecognized command rather than
				// hanging; STARTTLS lands here since this server never
				// advertised it.
				tp.PrintfLine("502 5.5.1 command not recognized")
			}
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return ln.Addr().String()
}

// fakeSMTPServerImplicitTLS is fakeSMTPServer wrapped so every accepted
// connection is TLS from the first byte, matching --smtp-require-tls.
func fakeSMTPServerImplicitTLS(t *testing.T, cert tls.Certificate) (addr string, dataCh <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	tlsLn := tls.NewListener(ln, &tls.Config{Certificates: []tls.Certificate{cert}})
	ch := make(chan string, 1)
	go func() {
		conn, err := tlsLn.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		tp := textproto.NewConn(conn)
		tp.PrintfLine("220 fake.local ESMTP")
		for {
			line, err := tp.ReadLine()
			if err != nil {
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
				tp.PrintfLine("250 fake.local")
			case strings.HasPrefix(line, "MAIL FROM"):
				tp.PrintfLine("250 OK")
			case strings.HasPrefix(line, "RCPT TO"):
				tp.PrintfLine("250 OK")
			case line == "DATA":
				tp.PrintfLine("354 go ahead")
				dr := tp.DotReader()
				data, _ := readAllBytes(dr)
				ch <- string(data)
				tp.PrintfLine("250 OK")
			case line == "QUIT":
				tp.PrintfLine("221 bye")
				return
			}
		}
	}()
	t.Cleanup(func() { tlsLn.Close() })
	return ln.Addr().String(), ch
}

func TestSendTLS_StartTLS_Success(t *testing.T) {
	cert := generateTestCert(t)
	addr, dataCh := fakeSMTPServerStartTLS(t, cert)

	spec := MessageSpec{From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi", Text: "hello starttls"}
	raw, err := spec.Build(time.Now())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	from, to := spec.Envelope()

	tlsOpts := TLSOptions{Mode: TLSStartTLS, InsecureSkipVerify: true}
	if err := SendTLS(context.Background(), addr, tlsOpts, nil, from, to, raw); err != nil {
		t.Fatalf("SendTLS: %v", err)
	}

	select {
	case data := <-dataCh:
		if !strings.Contains(data, "hello starttls") {
			t.Errorf("server received unexpected data: %q", data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for DATA")
	}
}

func TestSendTLS_StartTLS_NotAdvertised_ClearError(t *testing.T) {
	addr := fakeSMTPServerNoStartTLS(t)

	spec := MessageSpec{From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi", Text: "hello"}
	raw, err := spec.Build(time.Now())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	from, to := spec.Envelope()

	tlsOpts := TLSOptions{Mode: TLSStartTLS, InsecureSkipVerify: true}
	err = SendTLS(context.Background(), addr, tlsOpts, nil, from, to, raw)
	if err == nil {
		t.Fatal("expected an error when the server does not advertise STARTTLS, got nil")
	}
	if !strings.Contains(err.Error(), "STARTTLS") {
		t.Fatalf("expected error to mention STARTTLS, got: %v", err)
	}
}

func TestSendTLS_Implicit_Success(t *testing.T) {
	cert := generateTestCert(t)
	addr, dataCh := fakeSMTPServerImplicitTLS(t, cert)

	spec := MessageSpec{From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi", Text: "hello implicit"}
	raw, err := spec.Build(time.Now())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	from, to := spec.Envelope()

	tlsOpts := TLSOptions{Mode: TLSImplicit, InsecureSkipVerify: true}
	if err := SendTLS(context.Background(), addr, tlsOpts, nil, from, to, raw); err != nil {
		t.Fatalf("SendTLS: %v", err)
	}

	select {
	case data := <-dataCh:
		if !strings.Contains(data, "hello implicit") {
			t.Errorf("server received unexpected data: %q", data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for DATA")
	}
}

func TestSendTLS_Implicit_VerifyFailsWithoutInsecureSkipVerify(t *testing.T) {
	cert := generateTestCert(t)
	addr, _ := fakeSMTPServerImplicitTLS(t, cert)

	spec := MessageSpec{From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi", Text: "hello"}
	raw, err := spec.Build(time.Now())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	from, to := spec.Envelope()

	tlsOpts := TLSOptions{Mode: TLSImplicit, InsecureSkipVerify: false}
	err = SendTLS(context.Background(), addr, tlsOpts, nil, from, to, raw)
	if err == nil {
		t.Fatal("expected a certificate verification error, got nil")
	}
}

// TestSendTLS_PlaintextAgainstImplicitTLSServer_TimesOutClearly reproduces a
// real bug report: pointing a plaintext client (TLSNone, the default) at a
// --smtp-require-tls-style listener used to hang forever, since the server
// blocks reading a TLS ClientHello that a plaintext client never sends, and
// the client blocks reading a plaintext greeting the server never sends —
// neither side had anything bounding that mutual read. handshakeTimeout
// fixes this; shorten it here so the test doesn't take 30s.
func TestSendTLS_PlaintextAgainstImplicitTLSServer_TimesOutClearly(t *testing.T) {
	old := handshakeTimeout
	handshakeTimeout = 300 * time.Millisecond
	t.Cleanup(func() { handshakeTimeout = old })

	cert := generateTestCert(t)
	addr, _ := fakeSMTPServerImplicitTLS(t, cert)

	spec := MessageSpec{From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi", Text: "hello"}
	raw, err := spec.Build(time.Now())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	from, to := spec.Envelope()

	done := make(chan error, 1)
	go func() {
		done <- SendTLS(context.Background(), addr, TLSOptions{}, nil, from, to, raw)
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected an error when a plaintext client hits a TLS-only listener, got nil")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("SendTLS hung instead of timing out — handshakeTimeout did not bound the initial exchange")
	}
}

func TestParseTLSMode(t *testing.T) {
	tests := []struct {
		in      string
		want    TLSMode
		wantErr bool
	}{
		{"", TLSNone, false},
		{"none", TLSNone, false},
		{"starttls", TLSStartTLS, false},
		{"implicit", TLSImplicit, false},
		{"bogus", TLSNone, true},
	}
	for _, tt := range tests {
		got, err := ParseTLSMode(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseTLSMode(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
		}
		if got != tt.want {
			t.Errorf("ParseTLSMode(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func readAll(r interface{ Read([]byte) (int, error) }) (string, error) {
	var buf bytes.Buffer
	b := make([]byte, 4096)
	for {
		n, err := r.Read(b)
		buf.Write(b[:n])
		if err != nil {
			break
		}
	}
	return buf.String(), nil
}

func readAllBytes(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var buf bytes.Buffer
	b := make([]byte, 4096)
	for {
		n, err := r.Read(b)
		buf.Write(b[:n])
		if err != nil {
			break
		}
	}
	return buf.Bytes(), nil
}
