//go:build leakcheck

package ws

import (
	"testing"

	"go.uber.org/goleak"

	"github.com/0funct0ry/maelsink/internal/events"
)

// TestMain applies a goroutine-leak check across every test in this
// package, per SPEC.md §2.3 point 2: the WebSocket hub and event bus are
// explicitly called out as a package where per-client goroutines must not
// leak. Gated behind the "leakcheck" build tag so `make test` stays fast
// and `make test-leak` opts into this stricter check.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// TestHub_ManyClientsNoLeak connects and disconnects many WebSocket clients
// in sequence and asserts no subscriber/pump goroutines are left behind.
func TestHub_ManyClientsNoLeak(t *testing.T) {
	bus := events.NewBus()
	srv, hub := newTestServer(t, bus)

	for i := 0; i < 50; i++ {
		conn := dial(t, srv)
		readFrame(t, conn) // hello
		_ = conn.Close()
	}

	hub.Close()
}
