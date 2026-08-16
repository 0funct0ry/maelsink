package smtp

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"strings"
	"time"

	"github.com/0funct0ry/maelsink/internal/events"
	"github.com/0funct0ry/maelsink/internal/store"
)

// sessionState models transaction progress within a single connection, per
// RFC 5321's state machine: a client must HELO/EHLO before MAIL, MAIL before
// RCPT, and at least one RCPT before DATA.
type sessionState int

const (
	stateGreeted sessionState = iota // connected, no HELO/EHLO yet
	stateHelo                        // HELO/EHLO done, no transaction in progress
	stateMail                        // MAIL FROM accepted, awaiting RCPT
	stateRcpt                        // at least one RCPT TO accepted, DATA allowed
)

// commandLineTimeout bounds how long a session will wait for the next
// command line, so a client that connects and goes silent doesn't hold a
// handler goroutine (and its tracked connection) open forever.
const commandLineTimeout = 5 * time.Minute

// session holds the state for a single SMTP connection. One session is
// created per accepted net.Conn and is not shared across goroutines.
type session struct {
	srv  *Server
	conn net.Conn
	tp   *textproto.Reader
	w    *textproto.Writer

	state      sessionState
	heloDomain string
	authed     bool
	tlsActive  bool

	envFrom string
	envTo   []string

	// Session record + transcript (M8.4), persisted by Server.handleConn
	// once run() returns. See appendTranscript/finalize below.
	ID         string
	ClientIP   string
	ClientHELO string
	StartedAt  time.Time
	EndedAt    *time.Time
	Status     string
	MessageID  *string
	transcript []store.TranscriptLine

	// authMechanism is set for the duration of an in-progress AUTH
	// exchange (by handleAUTH, in auth.go) so readLine knows to redact
	// continuation lines it reads on AUTH's behalf, which bypass run()'s
	// main dispatch loop entirely.
	authMechanism string
}

func newSession(srv *Server, conn net.Conn) *session {
	sess := &session{srv: srv, state: stateGreeted, ID: store.NewID(), StartedAt: time.Now()}
	sess.ClientIP = clientIP(conn)
	sess.attach(conn)
	// In RequireTLS (implicit-TLS) mode, ListenAndServe already wraps every
	// accepted connection in a completed TLS handshake before handleConn
	// ever runs — so the session starts protected, with no STARTTLS
	// upgrade to perform.
	if srv.cfg.RequireTLS {
		sess.tlsActive = true
	}
	return sess
}

// clientIP returns conn's remote address with any port stripped, or the
// raw RemoteAddr string if it isn't a host:port pair.
func clientIP(conn net.Conn) string {
	addr := conn.RemoteAddr().String()
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

// appendTranscript records one wire line in the session's protocol
// transcript (M8.4). direction is 'C' (client -> server) or 'S' (server ->
// client). Callers pass the exact text as it appeared/was sent on the wire,
// already redacted where applicable (see readLine's AUTH handling).
//
// It also publishes a session.line event (M8.4a) for this entry, so a
// session's transcript can be live-tailed in the Web UI, and persists the
// line via AppendSessionLine immediately (rather than waiting for
// handleConn's final CreateSession call at connection close) — so a
// session opened mid-conversation shows everything captured so far on its
// very first fetch, not just lines that happen to arrive after the page
// was already open. A persist failure is logged, not fatal: it must never
// interrupt the SMTP transaction in progress.
func (sess *session) appendTranscript(direction byte, line string) {
	position := len(sess.transcript)
	tl := store.TranscriptLine{
		Direction: direction,
		Line:      line,
		Position:  position,
		Ts:        time.Now(),
	}
	sess.transcript = append(sess.transcript, tl)
	sess.srv.bus.Publish(events.SessionLine(sess.ID, direction, line, position))
	if err := sess.srv.store.AppendSessionLine(context.Background(), sess.ID, tl); err != nil {
		sess.srv.logger.Error("smtp: failed to persist session line", "session_id", sess.ID, "err", err)
	}
}

// appendDataLines records a DATA command's message body into the
// transcript as a sequence of 'C' entries, one per line. handleDATA reads
// the body via textproto.Reader.DotReader, which operates on the raw byte
// stream rather than sess.readLine's line-at-a-time interface (the one
// choke point every other client line goes through) — so without this, the
// entire message body would be silently absent from the transcript despite
// every other part of the conversation being captured.
//
// raw is exactly what DotReader returned: dot-unstuffed, and with every
// wire "\r\n" already normalized to a plain "\n" by net/textproto (see
// textproto.Reader.DotReader's doc comment) — appendDataLines splits on
// that, not "\r\n". This logs the reconstructed content rather than the
// literal dot-stuffed wire bytes — the right tradeoff for a human-readable
// debug transcript, since dot-stuffing is a transport escaping detail no
// one wants to see.
// terminated is true only when DotReader completed normally (a real "."
// line was seen and consumed) — appendDataLines then appends a synthetic
// "." entry for it, since DotReader never returns that line to callers.
// terminated is false for a read error or an oversized payload, where the
// connection is closed without ever seeing a terminator; the entries
// logged there are exactly the bytes received, with no implied line the
// wire didn't actually send.
func (sess *session) appendDataLines(raw []byte, terminated bool) {
	text := strings.TrimSuffix(string(raw), "\n")
	if text != "" {
		for _, line := range strings.Split(text, "\n") {
			sess.appendTranscript('C', line)
		}
	}
	if terminated {
		sess.appendTranscript('C', ".")
	}
}

// readLine reads one client line via the underlying textproto.Reader and
// records it in the transcript (redacted if it's an AUTH command or
// mid-exchange AUTH payload) before returning it. Every client-line read
// anywhere in the smtp package — run()'s main loop and auth.go's
// continuation reads alike — goes through this so no path can bypass
// transcript capture.
func (sess *session) readLine() (string, error) {
	line, err := sess.tp.ReadLine()
	if err != nil {
		return "", err
	}
	sess.appendClientLine(line)
	return line, nil
}

// appendClientLine redacts and records a raw client line. If an AUTH
// exchange is in progress (sess.authMechanism set), the line is a bare
// base64 credential blob and is stored as "<mechanism> [REDACTED]". A fresh
// "AUTH <mechanism> ..." command line is stored as "AUTH <mechanism>
// [REDACTED]", keeping the mechanism token but never the payload. Raw
// credential bytes are never persisted anywhere in the transcript.
func (sess *session) appendClientLine(line string) {
	if sess.authMechanism != "" {
		sess.appendTranscript('C', sess.authMechanism+" [REDACTED]")
		return
	}

	verb, arg := parseCommandLine(line)
	if verb == "AUTH" {
		mechanism := ""
		if fields := strings.Fields(arg); len(fields) > 0 {
			mechanism = strings.ToUpper(fields[0])
		}
		sess.appendTranscript('C', strings.TrimSpace("AUTH "+mechanism+" [REDACTED]"))
		return
	}

	sess.appendTranscript('C', line)
}

// finalize records the session's end time and terminal status. Called
// exactly once, from every exit path of run().
func (sess *session) finalize(status string) {
	now := time.Now()
	sess.EndedAt = &now
	sess.Status = status
}

// statusForReadErr classifies a readLine failure that ended run(): a
// deadlineReader timeout (an idle client) is "timeout", anything else
// (client disconnect, protocol-level read failure) is "aborted".
func statusForReadErr(err error) string {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}
	return "aborted"
}

// attach (re)wraps conn with fresh textproto reader/writer, used both at
// connection setup and after a successful STARTTLS upgrade.
func (sess *session) attach(conn net.Conn) {
	sess.conn = conn
	sess.tp = textproto.NewReader(bufio.NewReader(newDeadlineReader(conn, commandLineTimeout)))
	sess.w = textproto.NewWriter(newBufWriter(conn))
}

// run drives the session to completion: it sends the greeting, then reads
// and dispatches commands until a handler signals quit or the connection is
// no longer readable.
func (sess *session) run() {
	sess.reply(codeGreeting, fmt.Sprintf(msgGreetingFmt, sess.srv.cfg.Domain))

	for {
		line, err := sess.readLine()
		if err != nil {
			sess.finalize(statusForReadErr(err))
			return
		}

		verb, arg := parseCommandLine(line)
		if verb == "" {
			sess.reply(codeCommandUnrecognized, msgCommandUnrecognized)
			continue
		}

		if sess.dispatch(verb, arg) {
			// QUIT is the only verb whose handler ends the session by
			// design; every other quit=true return (a failed STARTTLS
			// handshake, an oversized/unreadable DATA payload) is an
			// abnormal end to the transaction.
			status := "rejected"
			if verb == "QUIT" {
				status = "completed"
			}
			sess.finalize(status)
			return
		}
	}
}

// parseCommandLine splits a command line into its verb (upper-cased) and
// the remainder of the line, per RFC 5321's "VERB [SP argument]" grammar
// (e.g. "MAIL FROM:<addr> SIZE=123" -> verb "MAIL", arg "FROM:<addr> SIZE=123").
func parseCommandLine(line string) (verb, arg string) {
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return "", ""
	}
	parts := strings.SplitN(line, " ", 2)
	verb = strings.ToUpper(parts[0])
	if len(parts) > 1 {
		arg = strings.TrimSpace(parts[1])
	}
	return verb, arg
}

// reply writes a single-line SMTP response: "<code> <message>\r\n".
func (sess *session) reply(code int, msg string) {
	line := fmt.Sprintf("%d %s", code, msg)
	sess.appendTranscript('S', line)
	_ = sess.w.PrintfLine("%s", line)
}

// replyMultiline writes an EHLO-style multi-line response: all but the last
// line use "<code>-<text>", the last line uses "<code> <text>".
func (sess *session) replyMultiline(code int, lines []string) {
	for i, text := range lines {
		sep := "-"
		if i == len(lines)-1 {
			sep = " "
		}
		line := fmt.Sprintf("%d%s%s", code, sep, text)
		sess.appendTranscript('S', line)
		_ = sess.w.PrintfLine("%s", line)
	}
}

// resetTransaction clears MAIL/RCPT state (RSET, and implicitly after a
// completed DATA or a successful STARTTLS handshake).
func (sess *session) resetTransaction() {
	sess.envFrom = ""
	sess.envTo = nil
}

// startTLS upgrades the connection in place, per RFC 3207: any HELO/MAIL/RCPT
// state must be discarded after the handshake, requiring the client to
// re-identify itself.
func (sess *session) startTLS() error {
	cert, err := tls.LoadX509KeyPair(sess.srv.cfg.TLSCert, sess.srv.cfg.TLSKey)
	if err != nil {
		return err
	}
	tlsConn := tls.Server(sess.conn, &tls.Config{Certificates: []tls.Certificate{cert}})
	if err := tlsConn.Handshake(); err != nil {
		return err
	}

	sess.srv.retrack(sess.conn, tlsConn)
	sess.attach(tlsConn)
	sess.tlsActive = true
	sess.state = stateGreeted
	sess.heloDomain = ""
	sess.authed = false
	sess.resetTransaction()
	return nil
}
