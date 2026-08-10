// Package ws implements maelsink's WebSocket hub (SPEC.md §5.5, M7.0):
// GET /ws upgrades a connection, registers it with the Hub, and the Hub
// fans out internal/events bus events to every connected client as JSON
// text frames.
package ws

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/0funct0ry/maelsink/internal/events"
)

// shutdownFlushWait bounds how long Shutdown waits for queued shutdown
// frames to be written before force-closing connections.
const shutdownFlushWait = 200 * time.Millisecond

var upgrader = websocket.Upgrader{
	// The Web UI SPA and the WS endpoint are always served from the same
	// origin (SPEC.md §5.5 / §3.4 base-path support); no cross-origin
	// WebSocket clients are expected.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub owns the set of live WebSocket clients and forwards every event
// published on bus to all of them. One Hub is constructed per process
// (cmd/serve.go) and passed into internal/webui.
type Hub struct {
	mu      sync.Mutex
	clients map[*client]struct{}

	bus    *events.Bus
	unsub  func()
	logger *slog.Logger
}

// NewHub constructs a Hub subscribed to bus and starts its single
// bus-fan-out goroutine. Call Close to stop that goroutine (used by tests
// that need a clean goroutine count; production callers let it run for the
// process lifetime).
func NewHub(bus *events.Bus, logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	h := &Hub{
		clients: make(map[*client]struct{}),
		bus:     bus,
		logger:  logger,
	}
	sub, unsub := bus.Subscribe()
	h.unsub = unsub
	go h.fanout(sub)
	return h
}

// fanout reads every event off sub and forwards it to all registered
// clients until sub is closed (i.e. until Close unsubscribes from bus).
func (h *Hub) fanout(sub <-chan events.Event) {
	for ev := range sub {
		f := eventFrame(ev)
		h.mu.Lock()
		for c := range h.clients {
			c.enqueueFrame(f)
		}
		h.mu.Unlock()
	}
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

// ServeWS upgrades the request to a WebSocket connection, registers it,
// sends the hello frame, and blocks (running the client's read/write pumps)
// until the connection closes. This is the GET /ws handler itself.
func (h *Hub) ServeWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Debug("ws: upgrade failed", "error", err)
		return
	}

	cl := newClient(conn)
	h.register(cl)
	defer h.unregister(cl)

	cl.enqueueFrame(helloFrame())

	go cl.writePump()
	cl.readPump() // blocks until the connection closes
}

// Shutdown broadcasts {"type":"server.shutdown"} to every currently
// connected client and closes their connections. It does not stop Hub from
// accepting new registrations — callers (cmd/serve.go) should stop the HTTP
// server's listener first/concurrently. Full graceful-drain coordination is
// M10.0's job; this is the "basic mechanism" M7.0 requires.
func (h *Hub) Shutdown(ctx context.Context) {
	h.mu.Lock()
	clients := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()

	f := shutdownFrame()
	for _, c := range clients {
		c.enqueueFrame(f)
	}
	// Give each client's write pump a brief chance to flush the shutdown
	// frame before force-closing the connection.
	flushDeadline := time.NewTimer(shutdownFlushWait)
	defer flushDeadline.Stop()
	select {
	case <-ctx.Done():
	case <-flushDeadline.C:
	}
	for _, c := range clients {
		c.close()
	}
}

// Close stops the Hub's bus fan-out goroutine. Intended for tests that
// assert no goroutines leak across many connect/disconnect cycles;
// production code lets the Hub live for the process.
func (h *Hub) Close() {
	h.unsub()
}
