package cmd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

// generateSelfSignedCert writes a throwaway self-signed PEM cert/key pair to
// dir, returning their paths, for exercising Web UI HTTPS without a
// real-world CA.
func generateSelfSignedCert(t *testing.T, dir string) (certPath, keyPath string) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")

	certOut, err := os.Create(certPath)
	if err != nil {
		t.Fatalf("create cert file: %v", err)
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatalf("encode cert: %v", err)
	}

	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	keyOut, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("create key file: %v", err)
	}
	defer keyOut.Close()
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		t.Fatalf("encode key: %v", err)
	}

	return certPath, keyPath
}

// freePort asks the OS for an ephemeral port and immediately releases it, so
// the caller can pass a concrete --*-port flag without a fixed-port
// collision across parallel test runs.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// waitForListen polls addr until a TCP connection succeeds or timeout
// elapses.
func waitForListen(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s to accept connections", addr)
}

// TestServe_Headless verifies M8.0's Definition of Done: --headless binds
// exactly the SMTP and REST API listeners, and the Web UI port is genuinely
// unbound (connection refused), not merely 404ing.
func TestServe_Headless(t *testing.T) {
	resetFlags(rootCmd)
	t.Cleanup(func() { resetFlags(rootCmd) })

	smtpPort := freePort(t)
	apiPort := freePort(t)
	webPort := freePort(t) // never bound; only used to prove nothing listens here

	dbPath := filepath.Join(t.TempDir(), "maelsink.db")

	rootCmd.SetArgs([]string{
		"serve",
		"--headless",
		"--smtp-host", "127.0.0.1",
		"--smtp-port", strconv.Itoa(smtpPort),
		"--api-host", "127.0.0.1",
		"--api-port", strconv.Itoa(apiPort),
		"--web-host", "127.0.0.1",
		"--web-port", strconv.Itoa(webPort),
		"--db", dbPath,
	})

	done := make(chan error, 1)
	go func() { done <- rootCmd.Execute() }()
	t.Cleanup(func() {
		_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("serve did not shut down in time")
		}
		rootCmd.SetArgs(nil)
	})

	waitForListen(t, fmt.Sprintf("127.0.0.1:%d", apiPort), 3*time.Second)

	if conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", webPort), 500*time.Millisecond); err == nil {
		conn.Close()
		t.Fatalf("expected connection refused on web port %d in headless mode, but it accepted a connection", webPort)
	}
}

// TestServe_WebTLS verifies M8.9's Definition of Done: setting
// --web-tls-cert/--web-tls-key serves the Web UI over HTTPS.
func TestServe_WebTLS(t *testing.T) {
	resetFlags(rootCmd)
	t.Cleanup(func() { resetFlags(rootCmd) })

	certPath, keyPath := generateSelfSignedCert(t, t.TempDir())

	smtpPort := freePort(t)
	apiPort := freePort(t)
	webPort := freePort(t)

	dbPath := filepath.Join(t.TempDir(), "maelsink.db")

	rootCmd.SetArgs([]string{
		"serve",
		"--smtp-host", "127.0.0.1",
		"--smtp-port", strconv.Itoa(smtpPort),
		"--api-host", "127.0.0.1",
		"--api-port", strconv.Itoa(apiPort),
		"--web-host", "127.0.0.1",
		"--web-port", strconv.Itoa(webPort),
		"--web-tls-cert", certPath,
		"--web-tls-key", keyPath,
		"--db", dbPath,
	})

	done := make(chan error, 1)
	go func() { done <- rootCmd.Execute() }()
	t.Cleanup(func() {
		_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("serve did not shut down in time")
		}
		rootCmd.SetArgs(nil)
	})

	waitForListen(t, fmt.Sprintf("127.0.0.1:%d", webPort), 3*time.Second)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 3 * time.Second,
	}
	resp, err := client.Get(fmt.Sprintf("https://127.0.0.1:%d/", webPort))
	if err != nil {
		t.Fatalf("GET over https: %v", err)
	}
	defer resp.Body.Close()
	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatalf("read body: %v", err)
	}

	plainResp, err := (&http.Client{Timeout: time.Second}).Get(fmt.Sprintf("http://127.0.0.1:%d/", webPort))
	if err == nil {
		defer plainResp.Body.Close()
		if plainResp.StatusCode < 400 {
			t.Fatalf("expected plain HTTP request to fail once TLS is enabled, got status %d", plainResp.StatusCode)
		}
	}
}

// TestServe_SMTPAuthEnabledRequiresACredentialSource verifies M8.10's
// fail-fast check: --smtp-auth-enabled with none of username/password,
// --smtp-auth-file, --smtp-auth-accept-any, or MAELSINK_SMTP_AUTH configured
// is rejected at startup rather than starting an AUTH mechanism nothing can
// ever satisfy.
func TestServe_SMTPAuthEnabledRequiresACredentialSource(t *testing.T) {
	resetFlags(rootCmd)
	t.Cleanup(func() { resetFlags(rootCmd) })

	dbPath := filepath.Join(t.TempDir(), "maelsink.db")

	rootCmd.SetArgs([]string{
		"serve",
		"--headless",
		"--smtp-host", "127.0.0.1",
		"--smtp-port", strconv.Itoa(freePort(t)),
		"--api-host", "127.0.0.1",
		"--api-port", strconv.Itoa(freePort(t)),
		"--smtp-auth-enabled",
		"--db", dbPath,
	})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	if err := rootCmd.Execute(); err == nil {
		t.Error("Execute() with smtp-auth-enabled and no credential source = nil, want error")
	}
}

// MAELSINK_SMTP_AUTH alone (with no username/password/file/accept_any) is a
// valid credential source and must not be rejected by the same check.
func TestServe_SMTPAuthEnabledAcceptsEnvOnlyCredentials(t *testing.T) {
	resetFlags(rootCmd)
	t.Cleanup(func() { resetFlags(rootCmd) })

	t.Setenv("MAELSINK_SMTP_AUTH", "envuser:envpass")

	smtpPort := freePort(t)
	apiPort := freePort(t)
	dbPath := filepath.Join(t.TempDir(), "maelsink.db")

	rootCmd.SetArgs([]string{
		"serve",
		"--headless",
		"--smtp-host", "127.0.0.1",
		"--smtp-port", strconv.Itoa(smtpPort),
		"--api-host", "127.0.0.1",
		"--api-port", strconv.Itoa(apiPort),
		"--smtp-auth-enabled",
		"--db", dbPath,
	})

	done := make(chan error, 1)
	go func() { done <- rootCmd.Execute() }()
	t.Cleanup(func() {
		_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("serve did not shut down in time")
		}
		rootCmd.SetArgs(nil)
	})

	waitForListen(t, fmt.Sprintf("127.0.0.1:%d", apiPort), 3*time.Second)
}
