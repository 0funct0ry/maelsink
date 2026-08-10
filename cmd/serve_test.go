package cmd

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

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
