package smtp

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/textproto"
	"strings"
	"time"
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
}

func newSession(srv *Server, conn net.Conn) *session {
	sess := &session{srv: srv, state: stateGreeted}
	sess.attach(conn)
	return sess
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
		line, err := sess.tp.ReadLine()
		if err != nil {
			return
		}

		verb, arg := parseCommandLine(line)
		if verb == "" {
			sess.reply(codeCommandUnrecognized, msgCommandUnrecognized)
			continue
		}

		if sess.dispatch(verb, arg) {
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
	_ = sess.w.PrintfLine("%d %s", code, msg)
}

// replyMultiline writes an EHLO-style multi-line response: all but the last
// line use "<code>-<text>", the last line uses "<code> <text>".
func (sess *session) replyMultiline(code int, lines []string) {
	for i, line := range lines {
		sep := "-"
		if i == len(lines)-1 {
			sep = " "
		}
		_ = sess.w.PrintfLine("%d%s%s", code, sep, line)
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
