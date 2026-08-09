package smtp

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0funct0ry/maelsink/internal/store"
)

func newTestServer(t *testing.T, cfg Config) (*Server, store.MessageStore) {
	t.Helper()
	cfg.Host = "127.0.0.1"
	cfg.Port = 0
	if cfg.Domain == "" {
		cfg.Domain = "maelsink.test"
	}
	if cfg.MaxMessageSize == 0 {
		cfg.MaxMessageSize = 1 << 20
	}

	st := store.NewMemoryStore()
	srv, err := New(cfg, st, store.NoopPublisher{}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.ListenAndServe(ctx) }()

	// Wait for the listener to actually be bound before returning.
	for i := 0; i < 100 && srv.Addr() == nil; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if srv.Addr() == nil {
		t.Fatal("server did not start listening")
	}

	t.Cleanup(func() {
		cancel()
		_ = srv.Close()
		<-done
	})

	return srv, st
}

func dialAndTransact(t *testing.T, addr string) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	tp := textproto.NewReader(bufio.NewReader(conn))
	if _, _, err := tp.ReadResponse(220); err != nil {
		t.Fatalf("greeting: %v", err)
	}

	cmds := []struct {
		cmd  string
		want int
	}{
		{"EHLO client.example.com", 250},
		{"MAIL FROM:<alice@example.com>", 250},
		{"RCPT TO:<bob@example.com>", 250},
	}
	for _, c := range cmds {
		if _, err := fmt.Fprintf(conn, "%s\r\n", c.cmd); err != nil {
			t.Fatalf("write %q: %v", c.cmd, err)
		}
		if _, _, err := tp.ReadResponse(c.want); err != nil {
			t.Fatalf("%s: %v", c.cmd, err)
		}
	}

	if _, err := fmt.Fprintf(conn, "DATA\r\n"); err != nil {
		t.Fatalf("write DATA: %v", err)
	}
	if _, _, err := tp.ReadResponse(354); err != nil {
		t.Fatalf("DATA: %v", err)
	}
	body := "Subject: integration test\r\n\r\nhello from the integration test\r\n.\r\n"
	if _, err := conn.Write([]byte(body)); err != nil {
		t.Fatalf("write body: %v", err)
	}
	if _, _, err := tp.ReadResponse(250); err != nil {
		t.Fatalf("post-DATA: %v", err)
	}

	if _, err := fmt.Fprintf(conn, "QUIT\r\n"); err != nil {
		t.Fatalf("write QUIT: %v", err)
	}
	if _, _, err := tp.ReadResponse(221); err != nil {
		t.Fatalf("QUIT: %v", err)
	}
}

func TestServer_EndToEndTransaction(t *testing.T) {
	srv, st := newTestServer(t, Config{})

	before := time.Now()
	dialAndTransact(t, srv.Addr().String())
	after := time.Now()

	msgs, total, err := st.List(context.Background(), store.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if !strings.Contains(msgs[0].TextBody, "hello from the integration test") {
		t.Errorf("TextBody = %q", msgs[0].TextBody)
	}
	if msgs[0].ReceivedAt.Before(before) || msgs[0].ReceivedAt.After(after) {
		t.Errorf("ReceivedAt = %v, want between %v and %v", msgs[0].ReceivedAt, before, after)
	}
}

func TestServer_ManySequentialConnections(t *testing.T) {
	srv, _ := newTestServer(t, Config{})
	addr := srv.Addr().String()

	for i := 0; i < 50; i++ {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			t.Fatalf("dial %d: %v", i, err)
		}
		tp := textproto.NewReader(bufio.NewReader(conn))
		if _, _, err := tp.ReadResponse(220); err != nil {
			t.Fatalf("greeting %d: %v", i, err)
		}
		if _, err := fmt.Fprintf(conn, "QUIT\r\n"); err != nil {
			t.Fatalf("write QUIT %d: %v", i, err)
		}
		if _, _, err := tp.ReadResponse(221); err != nil {
			t.Fatalf("QUIT %d: %v", i, err)
		}
		conn.Close()
	}
}

func TestServer_ConcurrentConnections(t *testing.T) {
	srv, st := newTestServer(t, Config{})
	addr := srv.Addr().String()

	const n = 10
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			dialAndTransact(t, addr)
		}()
	}
	wg.Wait()

	_, total, err := st.List(context.Background(), store.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != n {
		t.Fatalf("total = %d, want %d", total, n)
	}
}
