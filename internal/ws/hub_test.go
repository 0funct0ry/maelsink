package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/0funct0ry/maelsink/internal/events"
)

func newTestServer(t *testing.T, bus *events.Bus) (*httptest.Server, *Hub) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	hub := NewHub(bus, slog.New(slog.DiscardHandler))
	engine := gin.New()
	engine.GET("/ws", hub.ServeWS)
	srv := httptest.NewServer(engine)
	t.Cleanup(func() {
		hub.Close()
		srv.Close()
	})
	return srv, hub
}

func dial(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func readFrame(t *testing.T, conn *websocket.Conn) frame {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	var f frame
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("unmarshal frame: %v (data=%s)", err, data)
	}
	return f
}

func TestHub_SendsHelloOnConnect(t *testing.T) {
	bus := events.NewBus()
	srv, _ := newTestServer(t, bus)
	conn := dial(t, srv)

	f := readFrame(t, conn)
	if f.Type != "hello" {
		t.Fatalf("got type %q, want %q", f.Type, "hello")
	}
	payload, ok := f.Payload.(map[string]any)
	if !ok {
		t.Fatalf("payload is %T, want map", f.Payload)
	}
	if _, ok := payload["server_time"]; !ok {
		t.Fatal("hello payload missing server_time")
	}
	if _, ok := payload["version"]; !ok {
		t.Fatal("hello payload missing version")
	}
}

func TestHub_ForwardsMessageCreated(t *testing.T) {
	bus := events.NewBus()
	srv, _ := newTestServer(t, bus)
	conn := dial(t, srv)
	readFrame(t, conn) // hello

	bus.Publish(events.MessageCreated(map[string]string{"id": "msg_abc"}))

	f := readFrame(t, conn)
	if f.Type != string(events.TypeMessageCreated) {
		t.Fatalf("got type %q, want %q", f.Type, events.TypeMessageCreated)
	}
	payload, ok := f.Payload.(map[string]any)
	if !ok || payload["id"] != "msg_abc" {
		t.Fatalf("unexpected payload: %#v", f.Payload)
	}
}

func TestHub_ForwardsMessageDeleted(t *testing.T) {
	bus := events.NewBus()
	srv, _ := newTestServer(t, bus)
	conn := dial(t, srv)
	readFrame(t, conn) // hello

	bus.Publish(events.MessageDeleted("msg_123"))

	f := readFrame(t, conn)
	if f.Type != string(events.TypeMessageDeleted) {
		t.Fatalf("got type %q, want %q", f.Type, events.TypeMessageDeleted)
	}
	payload, ok := f.Payload.(map[string]any)
	if !ok || payload["id"] != "msg_123" {
		t.Fatalf("unexpected payload: %#v", f.Payload)
	}
}

func TestHub_ForwardsMessagesCleared(t *testing.T) {
	bus := events.NewBus()
	srv, _ := newTestServer(t, bus)
	conn := dial(t, srv)
	readFrame(t, conn) // hello

	bus.Publish(events.MessagesCleared())

	f := readFrame(t, conn)
	if f.Type != string(events.TypeMessagesCleared) {
		t.Fatalf("got type %q, want %q", f.Type, events.TypeMessagesCleared)
	}
	payload, ok := f.Payload.(map[string]any)
	if !ok || len(payload) != 0 {
		t.Fatalf("unexpected payload: %#v", f.Payload)
	}
}

func TestHub_BroadcastsToMultipleClients(t *testing.T) {
	bus := events.NewBus()
	srv, _ := newTestServer(t, bus)
	conn1 := dial(t, srv)
	conn2 := dial(t, srv)
	readFrame(t, conn1) // hello
	readFrame(t, conn2) // hello

	bus.Publish(events.MessagesCleared())

	f1 := readFrame(t, conn1)
	f2 := readFrame(t, conn2)
	if f1.Type != string(events.TypeMessagesCleared) || f2.Type != string(events.TypeMessagesCleared) {
		t.Fatalf("expected both clients to receive messages.cleared, got %q and %q", f1.Type, f2.Type)
	}
}

func TestHub_ShutdownBroadcastsAndCloses(t *testing.T) {
	bus := events.NewBus()
	srv, hub := newTestServer(t, bus)
	conn := dial(t, srv)
	readFrame(t, conn) // hello

	hub.Shutdown(context.Background())

	f := readFrame(t, conn)
	if f.Type != "server.shutdown" {
		t.Fatalf("got type %q, want %q", f.Type, "server.shutdown")
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("expected connection to be closed after shutdown")
	}
}
