// Package smtp implements maelsink's SMTP server (SPEC.md §4). It is fully
// isolated per SPEC.md §2.3 point 5: no imports of net/http, Gin, or any
// future REST/UI package — only the store.MessageStore interface and the
// store.Publisher event-bus stand-in. This lets the protocol layer be built
// and unit-tested independently of whether any HTTP surface exists.
package smtp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0funct0ry/maelsink/internal/store"
)

// Config holds the subset of the resolved application configuration the
// SMTP server needs. Callers (cmd/serve.go) build this from config.SMTP,
// converting MaxMessageSizeMB to bytes.
type Config struct {
	Host   string
	Port   int
	Domain string

	MaxMessageSize int64 // bytes

	StartTLS bool
	TLSCert  string
	TLSKey   string

	AuthEnabled  bool
	AuthUsername string
	AuthPassword string
}

// Server accepts SMTP connections and stores parsed messages via the
// injected MessageStore, publishing a Publish event after each save.
type Server struct {
	cfg       Config
	store     store.MessageStore
	publisher store.Publisher
	logger    *slog.Logger

	listenerMu sync.RWMutex
	listener   net.Listener
	conns      sync.Map // net.Conn -> struct{}, tracks live connections for Close
	wg         sync.WaitGroup
	closed     atomic.Bool
}

// New constructs a Server. It does not start listening; call ListenAndServe.
func New(cfg Config, st store.MessageStore, pub store.Publisher, logger *slog.Logger) (*Server, error) {
	if st == nil {
		return nil, fmt.Errorf("smtp: MessageStore must not be nil")
	}
	if pub == nil {
		pub = store.NoopPublisher{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{cfg: cfg, store: st, publisher: pub, logger: logger}, nil
}

// ListenAndServe binds the configured host:port and accepts connections
// until ctx is canceled or Close is called, whichever comes first. It
// blocks until the listener is fully stopped.
func (s *Server) ListenAndServe(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp: listen %s: %w", addr, err)
	}
	s.listenerMu.Lock()
	s.listener = ln
	s.listenerMu.Unlock()

	stopCh := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = s.Close()
		case <-stopCh:
		}
	}()
	defer close(stopCh)

	s.logger.Info("smtp: listening", "addr", ln.Addr().String())

	for {
		conn, err := ln.Accept()
		if err != nil {
			if s.closed.Load() {
				return nil
			}
			return fmt.Errorf("smtp: accept: %w", err)
		}

		s.track(conn)
		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

// Addr returns the listener's address. Only valid after ListenAndServe has
// started listening; intended for tests that bind port 0.
func (s *Server) Addr() net.Addr {
	s.listenerMu.RLock()
	defer s.listenerMu.RUnlock()
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

// Close stops accepting new connections, force-closes every tracked
// connection (unblocking any in-flight Read), and waits for all
// handleConn goroutines to exit before returning.
func (s *Server) Close() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}
	s.listenerMu.RLock()
	ln := s.listener
	s.listenerMu.RUnlock()
	if ln != nil {
		_ = ln.Close()
	}
	s.conns.Range(func(key, _ any) bool {
		_ = key.(net.Conn).Close()
		return true
	})
	s.wg.Wait()
	return nil
}

func (s *Server) track(conn net.Conn) {
	s.conns.Store(conn, struct{}{})
}

func (s *Server) untrack(conn net.Conn) {
	s.conns.Delete(conn)
}

// retrack swaps the tracked entry for a connection after STARTTLS replaces
// the underlying net.Conn with a *tls.Conn wrapping it.
func (s *Server) retrack(oldConn, newConn net.Conn) {
	s.conns.Delete(oldConn)
	s.conns.Store(newConn, struct{}{})
}

func (s *Server) handleConn(conn net.Conn) {
	defer s.wg.Done()
	defer func() {
		s.untrack(conn)
		_ = conn.Close()
	}()

	sess := newSession(s, conn)
	sess.run()
}

// deadlineReader enforces a read deadline before every Read call, so a
// session blocked on ReadLine/DotReader can't hold its handleConn goroutine
// (and tracked connection) open indefinitely against an idle client.
type deadlineReader struct {
	conn    net.Conn
	timeout time.Duration
}

func newDeadlineReader(conn net.Conn, timeout time.Duration) io.Reader {
	return &deadlineReader{conn: conn, timeout: timeout}
}

func (r *deadlineReader) Read(p []byte) (int, error) {
	_ = r.conn.SetReadDeadline(time.Now().Add(r.timeout))
	return r.conn.Read(p)
}

func newBufWriter(w io.Writer) *bufio.Writer {
	return bufio.NewWriter(w)
}
