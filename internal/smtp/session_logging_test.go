package smtp

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/0funct0ry/maelsink/internal/events"
	"github.com/0funct0ry/maelsink/internal/store"
)

// waitForSessionStatus blocks until sess.run() (launched by
// newTestSessionRaw) has returned — synchronizing on the done channel
// rather than polling sess's fields directly, which would race with
// run()'s goroutine still writing them — then returns the finalized status.
func waitForSessionStatus(t *testing.T, sess *session, done <-chan struct{}) string {
	t.Helper()
	select {
	case <-done:
		return sess.Status
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for session to finalize")
		return ""
	}
}

func newTestSessionRaw(t *testing.T, cfg Config) (*testClient, *session, store.MessageStore, *events.Bus, <-chan struct{}) {
	t.Helper()
	st := store.NewMemoryStore()
	bus := events.NewBus()
	srv := &Server{cfg: cfg, store: st, bus: bus, logger: slog.New(slog.DiscardHandler)}

	clientConn, serverConn := net.Pipe()
	sess := newSession(srv, serverConn)
	done := make(chan struct{})
	go func() {
		sess.run()
		close(done)
	}()

	return &testClient{t: t, conn: clientConn, r: bufio.NewReader(clientConn)}, sess, st, bus, done
}

// TestSession_TranscriptOrderingMatchesWireOrder verifies C/S interleaving
// in the recorded transcript matches the actual order lines crossed the
// wire, for a full happy-path transaction.
func TestSession_TranscriptOrderingMatchesWireOrder(t *testing.T) {
	c, sess, _, _, done := newTestSessionRaw(t, testConfig())
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

	waitForSessionStatus(t, sess, done)

	if sess.Status != "completed" {
		t.Fatalf("Status = %q, want completed", sess.Status)
	}
	if sess.EndedAt == nil {
		t.Fatal("EndedAt not set")
	}

	transcript := sess.transcript
	if len(transcript) == 0 {
		t.Fatal("transcript is empty")
	}
	// Position must be strictly increasing and match each entry's index.
	for i, line := range transcript {
		if line.Position != i {
			t.Fatalf("transcript[%d].Position = %d, want %d", i, line.Position, i)
		}
	}
	// First line is the server's greeting; second is the client's EHLO.
	if transcript[0].Direction != 'S' || !strings.Contains(transcript[0].Line, "220") {
		t.Fatalf("transcript[0] = %+v, want server greeting", transcript[0])
	}
	if transcript[1].Direction != 'C' || transcript[1].Line != "EHLO client.example.com" {
		t.Fatalf("transcript[1] = %+v, want client EHLO", transcript[1])
	}
	// Last line before finalize is the server's closing reply.
	last := transcript[len(transcript)-1]
	if last.Direction != 'S' || !strings.Contains(last.Line, "221") {
		t.Fatalf("last transcript entry = %+v, want closing reply", last)
	}
}

// TestSession_AuthPlainInlineRedacted verifies AUTH PLAIN's inline base64
// argument is redacted (mechanism kept) rather than stored raw.
func TestSession_AuthPlainInlineRedacted(t *testing.T) {
	cfg := testConfig()
	cfg.AuthEnabled = true
	cfg.AuthUsername = "user"
	cfg.AuthPassword = "pass"

	c, sess, _, _, done := newTestSessionRaw(t, cfg)
	defer c.conn.Close()

	c.expectCode(codeGreeting)
	c.send("EHLO client.example.com")
	c.expectCode(codeOK)

	blob := base64.StdEncoding.EncodeToString([]byte("\x00user\x00pass"))
	c.send("AUTH PLAIN " + blob)
	c.expectCode(codeAuthSuccess)

	c.send("QUIT")
	c.expectCode(codeClosing)
	waitForSessionStatus(t, sess, done)

	var authLine *store.TranscriptLine
	for i := range sess.transcript {
		if sess.transcript[i].Direction == 'C' && strings.HasPrefix(sess.transcript[i].Line, "AUTH") {
			authLine = &sess.transcript[i]
		}
	}
	if authLine == nil {
		t.Fatal("no AUTH line found in transcript")
	}
	if authLine.Line != "AUTH PLAIN [REDACTED]" {
		t.Fatalf("AUTH transcript line = %q, want redacted with mechanism kept", authLine.Line)
	}
	for _, line := range sess.transcript {
		if strings.Contains(line.Line, blob) {
			t.Fatalf("raw AUTH payload leaked into transcript: %q", line.Line)
		}
	}
}

// TestSession_AuthLoginMultiStepRedacted verifies a full AUTH LOGIN
// challenge/response exchange (mechanism decided up front, two
// continuation lines read outside run()'s main loop) is fully redacted.
func TestSession_AuthLoginMultiStepRedacted(t *testing.T) {
	cfg := testConfig()
	cfg.AuthEnabled = true
	cfg.AuthUsername = "user"
	cfg.AuthPassword = "pass"

	c, sess, _, _, done := newTestSessionRaw(t, cfg)
	defer c.conn.Close()

	c.expectCode(codeGreeting)
	c.send("EHLO client.example.com")
	c.expectCode(codeOK)

	c.send("AUTH LOGIN")
	c.expectCode(codeAuthContinue)
	c.send(base64.StdEncoding.EncodeToString([]byte("user")))
	c.expectCode(codeAuthContinue)
	c.send(base64.StdEncoding.EncodeToString([]byte("pass")))
	c.expectCode(codeAuthSuccess)

	c.send("QUIT")
	c.expectCode(codeClosing)
	waitForSessionStatus(t, sess, done)

	var clientLines []string
	for _, line := range sess.transcript {
		if line.Direction == 'C' {
			clientLines = append(clientLines, line.Line)
		}
	}
	// EHLO, AUTH LOGIN, username continuation, password continuation, QUIT.
	found := 0
	for _, l := range clientLines {
		if l == "LOGIN [REDACTED]" {
			found++
		}
		if strings.Contains(l, "user") && l != "EHLO client.example.com" {
			t.Fatalf("raw username leaked into transcript: %q", l)
		}
		if strings.Contains(l, "pass") {
			t.Fatalf("raw password leaked into transcript: %q", l)
		}
	}
	if found != 2 {
		t.Fatalf("expected 2 redacted LOGIN continuation lines, found %d (lines: %v)", found, clientLines)
	}
	if clientLines[1] != "AUTH LOGIN [REDACTED]" {
		t.Fatalf("AUTH command line = %q, want redacted with mechanism kept", clientLines[1])
	}
}

// TestSession_FinalizeStatusAborted verifies an ungraceful client
// disconnect (no QUIT) finalizes as "aborted".
func TestSession_FinalizeStatusAborted(t *testing.T) {
	c, sess, _, _, done := newTestSessionRaw(t, testConfig())

	c.expectCode(codeGreeting)
	c.send("EHLO client.example.com")
	c.expectCode(codeOK)
	c.conn.Close()

	status := waitForSessionStatus(t, sess, done)
	if status != "aborted" {
		t.Fatalf("Status = %q, want aborted", status)
	}
}

// TestSession_FinalizeStatusRejected verifies a failed STARTTLS handshake
// (a mid-protocol handler-initiated quit that isn't QUIT) finalizes as
// "rejected".
func TestSession_FinalizeStatusRejected(t *testing.T) {
	cfg := testConfig()
	cfg.TLSCert = "/nonexistent/cert.pem"
	cfg.TLSKey = "/nonexistent/key.pem"

	c, sess, _, _, done := newTestSessionRaw(t, cfg)
	defer c.conn.Close()

	c.expectCode(codeGreeting)
	c.send("EHLO client.example.com")
	c.expectCode(codeOK)
	c.send("STARTTLS")
	c.expectCode(codeGreeting)

	status := waitForSessionStatus(t, sess, done)
	if status != "rejected" {
		t.Fatalf("Status = %q, want rejected", status)
	}
}

// TestServer_PersistsSessionAndPublishesEvents is a full integration test
// (real TCP socket, through Server.handleConn) verifying a finished session
// is persisted via CreateSession and both session.started/session.completed
// fire on the bus, cross-linked to the message it produced.
func TestServer_PersistsSessionAndPublishesEvents(t *testing.T) {
	srv, st := newTestServer(t, Config{})

	sub, unsub := srv.bus.Subscribe()
	defer unsub()

	dialAndTransact(t, srv.Addr().String())

	var started, completed events.Event
	deadline := time.After(2 * time.Second)
	for started.Type == "" || completed.Type == "" {
		select {
		case ev := <-sub:
			switch ev.Type {
			case events.TypeSessionStarted:
				started = ev
			case events.TypeSessionCompleted:
				completed = ev
			}
		case <-deadline:
			t.Fatalf("timed out waiting for session events (started=%v completed=%v)", started.Type, completed.Type)
		}
	}

	msgs, _, err := st.List(context.Background(), store.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].SessionID == "" {
		t.Fatal("message.SessionID not set")
	}

	sess, err := st.GetSession(context.Background(), msgs[0].SessionID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if sess.Status != "completed" {
		t.Fatalf("session Status = %q, want completed", sess.Status)
	}
	if sess.MessageID == nil || *sess.MessageID != msgs[0].ID {
		t.Fatalf("session MessageID = %v, want %q", sess.MessageID, msgs[0].ID)
	}
	if len(sess.Transcript) == 0 {
		t.Fatal("persisted session has no transcript")
	}
}

// TestSession_PublishesSessionLineEvents verifies each transcript entry
// also publishes a session.line event (M8.4a), in the same order and with
// matching direction/line/position — the mechanism live-tailing a session
// detail screen relies on before the connection closes.
func TestSession_PublishesSessionLineEvents(t *testing.T) {
	c, sess, _, bus, done := newTestSessionRaw(t, testConfig())
	defer c.conn.Close()

	sub, unsub := bus.Subscribe()
	defer unsub()

	c.expectCode(codeGreeting)
	c.send("EHLO client.example.com")
	c.expectCode(codeOK)
	c.send("QUIT")
	c.expectCode(codeClosing)
	waitForSessionStatus(t, sess, done)

	var lineEvents []events.Event
	deadline := time.After(2 * time.Second)
collect:
	for {
		select {
		case ev := <-sub:
			if ev.Type == events.TypeSessionLine {
				lineEvents = append(lineEvents, ev)
			}
			if len(lineEvents) >= len(sess.transcript) {
				break collect
			}
		case <-deadline:
			t.Fatalf("timed out collecting session.line events, got %d of %d", len(lineEvents), len(sess.transcript))
		}
	}

	if len(lineEvents) != len(sess.transcript) {
		t.Fatalf("got %d session.line events, want %d (one per transcript entry)", len(lineEvents), len(sess.transcript))
	}
	for i, ev := range lineEvents {
		want := sess.transcript[i]
		got := decodeSessionLinePayload(t, ev.Payload)
		if got.SessionID != sess.ID {
			t.Fatalf("event[%d].session_id = %q, want %q", i, got.SessionID, sess.ID)
		}
		if got.Direction != string(want.Direction) {
			t.Fatalf("event[%d].direction = %q, want %q", i, got.Direction, string(want.Direction))
		}
		if got.Line != want.Line {
			t.Fatalf("event[%d].line = %q, want %q", i, got.Line, want.Line)
		}
		if got.Position != want.Position {
			t.Fatalf("event[%d].position = %d, want %d", i, got.Position, want.Position)
		}
	}
}

// sessionLineFields mirrors internal/events' unexported sessionLinePayload
// JSON shape, so this test (outside that package) can assert on it.
type sessionLineFields struct {
	SessionID string `json:"session_id"`
	Direction string `json:"direction"`
	Line      string `json:"line"`
	Position  int    `json:"position"`
}

func decodeSessionLinePayload(t *testing.T, payload any) sessionLineFields {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshaling session.line payload: %v", err)
	}
	var fields sessionLineFields
	if err := json.Unmarshal(b, &fields); err != nil {
		t.Fatalf("unmarshaling session.line payload: %v", err)
	}
	return fields
}

// TestSession_TranscriptIncludesDataBody is a regression test: the DATA
// command's message body is read via textproto.Reader.DotReader directly
// (commands.go), bypassing sess.readLine's one choke point every other
// client line goes through — without appendDataLines wiring it in, the
// entire message body was silently missing from the transcript despite
// every other part of the conversation being captured.
func TestSession_TranscriptIncludesDataBody(t *testing.T) {
	c, sess, _, _, done := newTestSessionRaw(t, testConfig())
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
	waitForSessionStatus(t, sess, done)

	var clientLines []string
	for _, line := range sess.transcript {
		if line.Direction == 'C' {
			clientLines = append(clientLines, line.Line)
		}
	}
	want := []string{
		"EHLO client.example.com",
		"MAIL FROM:<alice@example.com>",
		"RCPT TO:<bob@example.com>",
		"DATA",
		"Subject: hi",
		"",
		"hello world",
		".",
		"QUIT",
	}
	if len(clientLines) != len(want) {
		t.Fatalf("client transcript lines = %q, want %q", clientLines, want)
	}
	for i, w := range want {
		if clientLines[i] != w {
			t.Fatalf("clientLines[%d] = %q, want %q (full: %q)", i, clientLines[i], w, clientLines)
		}
	}
}

// TestSession_TranscriptOmitsTerminatorForOversizedData verifies the
// oversized-DATA path (which closes the connection before a real "."
// terminator is ever seen) doesn't fabricate one in the transcript, while
// still logging what was actually received.
func TestSession_TranscriptOmitsTerminatorForOversizedData(t *testing.T) {
	cfg := testConfig()
	cfg.MaxMessageSize = 10
	c, sess, _, _, done := newTestSessionRaw(t, cfg)
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
	c.send("this line is definitely longer than ten bytes")
	c.expectCode(codeExceededStorage)

	waitForSessionStatus(t, sess, done)

	for _, line := range sess.transcript {
		if line.Direction == 'C' && line.Line == "." {
			t.Fatalf("transcript contains a fabricated terminator \".\" entry despite no real terminator being seen: %+v", sess.transcript)
		}
	}
}

// TestServer_SessionTranscriptVisibleBeforeCompletion is a regression test
// (M8.4a follow-up): fetching an in-progress session's transcript from the
// store — the same path GET /api/v1/sessions/{id} uses — must return every
// line captured so far, not just what was there at connection close. Before
// this fix, session_lines were only ever written once, in handleConn's
// final CreateSession call, so opening a Session Detail page mid-connection
// showed nothing until the connection closed.
func TestServer_SessionTranscriptVisibleBeforeCompletion(t *testing.T) {
	srv, st := newTestServer(t, Config{})

	conn, err := net.DialTimeout("tcp", srv.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	tp := textproto.NewReader(bufio.NewReader(conn))
	if _, _, err := tp.ReadResponse(220); err != nil {
		t.Fatalf("greeting: %v", err)
	}
	if _, err := fmt.Fprintf(conn, "EHLO client.example.com\r\n"); err != nil {
		t.Fatalf("write EHLO: %v", err)
	}
	if _, _, err := tp.ReadResponse(250); err != nil {
		t.Fatalf("EHLO: %v", err)
	}

	// Query the store now, before QUIT is ever sent — the connection is
	// still open. This is exactly what opening the Session Detail page
	// mid-conversation does.
	sessions, total, err := st.ListSessions(context.Background(), store.SessionListFilter{})
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if total != 1 {
		t.Fatalf("total sessions = %d, want 1", total)
	}

	full, err := st.GetSession(context.Background(), sessions[0].ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if full.Status != "" {
		t.Fatalf("Status = %q, want empty (still in progress)", full.Status)
	}

	var clientLines []string
	for _, line := range full.Transcript {
		if line.Direction == 'C' {
			clientLines = append(clientLines, line.Line)
		}
	}
	if len(clientLines) != 1 || clientLines[0] != "EHLO client.example.com" {
		t.Fatalf("client transcript lines fetched mid-session = %q, want [\"EHLO client.example.com\"] (the greeting/EHLO-reply lines should already be persisted too)", clientLines)
	}
	if len(full.Transcript) < 2 {
		t.Fatalf("Transcript = %+v, want at least the server greeting and EHLO reply already persisted", full.Transcript)
	}

	if _, err := fmt.Fprintf(conn, "QUIT\r\n"); err != nil {
		t.Fatalf("write QUIT: %v", err)
	}
	if _, _, err := tp.ReadResponse(221); err != nil {
		t.Fatalf("QUIT: %v", err)
	}
}
