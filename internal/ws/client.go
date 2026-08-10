package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// sendBufSize is the per-client outbound frame queue. A full queue
	// means the client is slow/stuck; the hub drops the frame for that
	// client rather than blocking the fan-out goroutine that serves every
	// other client.
	sendBufSize = 64

	// pingInterval is how often the server sends a ping control frame to
	// keep the connection alive and detect dead clients.
	pingInterval = 30 * time.Second
	// pongWait is how long to wait for a pong (or any other read activity)
	// before considering the connection dead.
	pongWait = 60 * time.Second
	// writeWait bounds how long a single WriteMessage/WriteControl call may
	// block.
	writeWait = 10 * time.Second
)

// client represents one connected WebSocket peer. All three of its
// goroutines (writePump, readPump, and its bus fan-out delivery via send)
// exit once done is closed, which happens exactly once via closeOnce.
type client struct {
	conn *websocket.Conn
	send chan []byte

	done      chan struct{}
	closeOnce sync.Once
}

func newClient(conn *websocket.Conn) *client {
	return &client{
		conn: conn,
		send: make(chan []byte, sendBufSize),
		done: make(chan struct{}),
	}
}

// enqueue attempts to queue payload for delivery, dropping it (rather than
// blocking) if the client's send buffer is full or it has already closed.
func (c *client) enqueue(payload []byte) {
	select {
	case c.send <- payload:
	case <-c.done:
	default:
	}
}

// enqueueFrame marshals f and enqueues it.
func (c *client) enqueueFrame(f frame) {
	b, err := json.Marshal(f)
	if err != nil {
		return
	}
	c.enqueue(b)
}

// close marks the client as done and closes the underlying connection.
// Safe to call more than once and from multiple goroutines.
func (c *client) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

// writePump owns every write to conn (gorilla/websocket requires a single
// writer goroutine per connection). It exits when done is closed, either
// because readPump detected a closed/broken connection or because the hub
// is shutting the client down.
func (c *client) writePump() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case msg := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				c.close()
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.close()
				return
			}
		}
	}
}

// readPump's only job is to detect the connection closing (no client->server
// frames are expected per SPEC.md §5.5) and to keep the read deadline
// advancing via the pong handler. It blocks until the connection errors or
// closes, then closes the client.
func (c *client) readPump() {
	defer c.close()

	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}
