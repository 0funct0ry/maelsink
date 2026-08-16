// Package cliclient holds the shared logic behind maelsink's CLI client
// commands (SPEC.md §7.2-7.3): building and sending a test message over SMTP
// (`maelsink send`), and a thin REST API client + table renderer for the
// read/delete commands. Kept out of /cmd so it can be unit-tested directly.
package cliclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"time"
)

// AttachmentSpec is one entry in MessageSpec.Attachments, or built from
// --attach flags.
type AttachmentSpec struct {
	Path     string `json:"path"`
	Filename string `json:"filename,omitempty"`
}

// MessageSpec is the message to send, populated either from CLI flags or
// unmarshaled from a --file JSON document. This JSON shape is maelsink's own
// definition (SPEC.md §7.2 only specifies that --file reads "a JSON message
// spec" without pinning down the fields) and mirrors the flag set 1:1.
type MessageSpec struct {
	From        string           `json:"from"`
	To          []string         `json:"to"`
	Cc          []string         `json:"cc,omitempty"`
	Bcc         []string         `json:"bcc,omitempty"`
	Subject     string           `json:"subject"`
	Text        string           `json:"text,omitempty"`
	HTML        string           `json:"html,omitempty"`
	Attachments []AttachmentSpec `json:"attachments,omitempty"`
	// Tags becomes one "X-Tag" header per entry — internal/smtp/mime.go's
	// extractTags derives internal/store.Message.Tags from that header on
	// ingest, so this is the client-side way to set tags on a sent message.
	Tags []string `json:"tags,omitempty"`
}

// Envelope returns the SMTP envelope (MAIL FROM / RCPT TO) for this spec.
func (m MessageSpec) Envelope() (from string, to []string) {
	to = make([]string, 0, len(m.To)+len(m.Cc)+len(m.Bcc))
	to = append(to, m.To...)
	to = append(to, m.Cc...)
	to = append(to, m.Bcc...)
	return m.From, to
}

// Build renders m into a raw RFC 5322 message, adding a multipart/mixed
// envelope only when attachments are present, and multipart/alternative for
// the body only when both Text and HTML are set.
func (m MessageSpec) Build(now time.Time) ([]byte, error) {
	var buf bytes.Buffer

	headers := textproto.MIMEHeader{}
	headers.Set("From", m.From)
	if len(m.To) > 0 {
		headers.Set("To", joinAddrs(m.To))
	}
	if len(m.Cc) > 0 {
		headers.Set("Cc", joinAddrs(m.Cc))
	}
	headers.Set("Subject", m.Subject)
	headers.Set("Date", now.UTC().Format(time.RFC1123Z))
	headers.Set("MIME-Version", "1.0")
	for _, tag := range m.Tags {
		headers.Add("X-Tag", tag)
	}

	body, bodyContentType, err := buildBody(m.Text, m.HTML)
	if err != nil {
		return nil, err
	}

	if len(m.Attachments) == 0 {
		headers.Set("Content-Type", bodyContentType)
		writeHeaders(&buf, headers)
		buf.Write(body)
		return buf.Bytes(), nil
	}

	mw := multipart.NewWriter(&buf)
	headers.Set("Content-Type", fmt.Sprintf(`multipart/mixed; boundary="%s"`, mw.Boundary()))
	writeHeaders(&buf, headers)

	bodyWriter, err := mw.CreatePart(textproto.MIMEHeader{"Content-Type": {bodyContentType}})
	if err != nil {
		return nil, fmt.Errorf("cliclient: create body part: %w", err)
	}
	if _, err := bodyWriter.Write(body); err != nil {
		return nil, fmt.Errorf("cliclient: write body part: %w", err)
	}

	for _, att := range m.Attachments {
		if err := attachFile(mw, att); err != nil {
			return nil, err
		}
	}

	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("cliclient: close multipart writer: %w", err)
	}

	return buf.Bytes(), nil
}

func buildBody(text, html string) (body []byte, contentType string, err error) {
	switch {
	case text != "" && html != "":
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		contentType = fmt.Sprintf(`multipart/alternative; boundary="%s"`, mw.Boundary())

		tw, err := mw.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain; charset=utf-8"}})
		if err != nil {
			return nil, "", err
		}
		if _, err := tw.Write([]byte(text)); err != nil {
			return nil, "", err
		}

		hw, err := mw.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/html; charset=utf-8"}})
		if err != nil {
			return nil, "", err
		}
		if _, err := hw.Write([]byte(html)); err != nil {
			return nil, "", err
		}

		if err := mw.Close(); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), contentType, nil
	case html != "":
		return []byte(html), "text/html; charset=utf-8", nil
	default:
		return []byte(text), "text/plain; charset=utf-8", nil
	}
}

// AttachFile writes att as one multipart part on mw. Exported so callers
// building their own multipart/mixed envelope outside MessageSpec.Build
// (e.g. internal/compose's EML-format /send, which splices attachments
// around an already-rendered whole-document template) can reuse the same
// attachment-encoding logic instead of duplicating it.
func AttachFile(mw *multipart.Writer, att AttachmentSpec) error {
	return attachFile(mw, att)
}

func attachFile(mw *multipart.Writer, att AttachmentSpec) error {
	data, err := os.ReadFile(att.Path)
	if err != nil {
		return fmt.Errorf("cliclient: read attachment %q: %w", att.Path, err)
	}
	filename := att.Filename
	if filename == "" {
		filename = filepath.Base(att.Path)
	}
	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	w, err := mw.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {contentType},
		"Content-Transfer-Encoding": {"base64"},
		"Content-Disposition":       {fmt.Sprintf(`attachment; filename="%s"`, filename)},
	})
	if err != nil {
		return fmt.Errorf("cliclient: create attachment part: %w", err)
	}
	if err := writeBase64(w, data); err != nil {
		return fmt.Errorf("cliclient: write attachment %q: %w", att.Path, err)
	}
	return nil
}

func writeBase64(w io.Writer, data []byte) error {
	enc := base64.NewEncoder(base64.StdEncoding, &lineWrapper{w: w, width: 76})
	if _, err := enc.Write(data); err != nil {
		return err
	}
	return enc.Close()
}

// lineWrapper inserts a CRLF every width bytes written, as required for
// base64-encoded MIME body parts.
type lineWrapper struct {
	w     io.Writer
	width int
	n     int
}

func (l *lineWrapper) Write(p []byte) (int, error) {
	written := 0
	for len(p) > 0 {
		space := l.width - l.n
		chunk := p
		if len(chunk) > space {
			chunk = chunk[:space]
		}
		n, err := l.w.Write(chunk)
		written += n
		l.n += n
		if err != nil {
			return written, err
		}
		p = p[n:]
		if l.n == l.width {
			if _, err := l.w.Write([]byte("\r\n")); err != nil {
				return written, err
			}
			l.n = 0
		}
	}
	return written, nil
}

func joinAddrs(addrs []string) string {
	out := ""
	for i, a := range addrs {
		if i > 0 {
			out += ", "
		}
		out += a
	}
	return out
}

func writeHeaders(buf *bytes.Buffer, headers textproto.MIMEHeader) {
	for _, key := range []string{"From", "To", "Cc", "Subject", "X-Tag", "Date", "MIME-Version", "Content-Type"} {
		for _, v := range headers.Values(key) {
			fmt.Fprintf(buf, "%s: %s\r\n", key, v)
		}
	}
	buf.WriteString("\r\n")
}

// Auth holds optional SMTP AUTH PLAIN credentials for Send.
type Auth struct {
	Username string
	Password string
}

// TLSMode selects how SendTLS establishes transport security with the SMTP
// server, mirroring the modes internal/smtp's server-side hardening (M8.10)
// supports: no TLS, opportunistic STARTTLS, or dialing straight into TLS.
type TLSMode int

const (
	// TLSNone dials plain TCP with no transport security. This is Send's
	// (and SendTLS's zero-value) behavior, unchanged from before M8.10a.
	TLSNone TLSMode = iota
	// TLSStartTLS dials plain TCP, then issues STARTTLS after EHLO,
	// upgrading the connection before AUTH/MAIL FROM/etc.
	TLSStartTLS
	// TLSImplicit dials directly into TLS (no STARTTLS negotiation),
	// matching a server started with --smtp-require-tls.
	TLSImplicit
)

// ParseTLSMode parses the --smtp-tls flag value ("none", "starttls", or
// "implicit") into a TLSMode, returning a clear error for anything else.
func ParseTLSMode(s string) (TLSMode, error) {
	switch s {
	case "", "none":
		return TLSNone, nil
	case "starttls":
		return TLSStartTLS, nil
	case "implicit":
		return TLSImplicit, nil
	default:
		return TLSNone, fmt.Errorf("cliclient: invalid --smtp-tls value %q (want none, starttls, or implicit)", s)
	}
}

// TLSOptions configures SendTLS's transport security.
type TLSOptions struct {
	Mode TLSMode
	// InsecureSkipVerify accepts a self-signed/dev certificate without
	// verifying it against a CA — expected routine use against maelsink's
	// own --smtp-tls-cert/--smtp-tls-key, never against a production server.
	InsecureSkipVerify bool
	// ServerName overrides the hostname used for SNI/certificate
	// verification. Empty uses the dial address's host.
	ServerName string
}

// handshakeTimeout bounds every step up through RCPT TO (dial, initial
// greeting, EHLO, optional STARTTLS, optional AUTH, MAIL FROM/RCPT TO) so
// that pointing a client at a mismatched TLS mode (e.g. a plaintext client
// against a --smtp-require-tls listener, which is waiting for a TLS
// ClientHello the client never sends) fails with a clear timeout instead of
// hanging forever — both sides would otherwise block on a read that never
// resolves. It is a package var, not a const, so tests can shorten it.
// Deliberately not applied to the DATA write below, since a large
// attachment can legitimately take longer than this to transfer once the
// TLS mode is confirmed correct.
var handshakeTimeout = 30 * time.Second

// Send dials addr, optionally authenticates, and delivers raw to the given
// envelope recipients. It returns the underlying *textproto.Error verbatim
// on rejection so callers can surface the server's exact error text.
//
// Send is a thin wrapper around SendTLS with TLSOptions{} (TLSNone),
// preserved so existing callers compile unchanged.
func Send(ctx context.Context, addr string, auth *Auth, from string, to []string, raw []byte) error {
	return SendTLS(ctx, addr, TLSOptions{}, auth, from, to, raw)
}

// SendTLS dials addr (optionally over TLS per tlsOpts), optionally
// authenticates, and delivers raw to the given envelope recipients.
func SendTLS(ctx context.Context, addr string, tlsOpts TLSOptions, auth *Auth, from string, to []string, raw []byte) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	serverName := tlsOpts.ServerName
	if serverName == "" {
		serverName = host
	}

	var conn net.Conn
	if tlsOpts.Mode == TLSImplicit {
		var d tls.Dialer
		d.Config = &tls.Config{InsecureSkipVerify: tlsOpts.InsecureSkipVerify, ServerName: serverName}
		conn, err = d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("cliclient: implicit TLS dial %s: %w", addr, err)
		}
	} else {
		var d net.Dialer
		conn, err = d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("cliclient: dial %s: %w", addr, err)
		}
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(handshakeTimeout)); err != nil {
		return fmt.Errorf("cliclient: set deadline: %w", err)
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("cliclient: smtp handshake: %w", err)
	}
	defer c.Close()

	if err := c.Hello("maelsink-cli"); err != nil {
		return fmt.Errorf("cliclient: EHLO: %w", err)
	}

	if tlsOpts.Mode == TLSStartTLS {
		if err := c.StartTLS(&tls.Config{InsecureSkipVerify: tlsOpts.InsecureSkipVerify, ServerName: serverName}); err != nil {
			return fmt.Errorf("cliclient: STARTTLS: %w", err)
		}
	}

	if auth != nil {
		if err := c.Auth(smtp.PlainAuth("", auth.Username, auth.Password, host)); err != nil {
			return fmt.Errorf("cliclient: AUTH: %w", err)
		}
	}

	if err := c.Mail(from); err != nil {
		return fmt.Errorf("cliclient: MAIL FROM: %w", err)
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("cliclient: RCPT TO %s: %w", rcpt, err)
		}
	}

	// Clear the handshake deadline before the potentially-large DATA
	// transfer: everything up to here is small protocol chatter that should
	// never legitimately take this long, but a big attachment might.
	if err := conn.SetDeadline(time.Time{}); err != nil {
		return fmt.Errorf("cliclient: clear deadline: %w", err)
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("cliclient: DATA: %w", err)
	}
	if _, err := w.Write(raw); err != nil {
		return fmt.Errorf("cliclient: write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("cliclient: finish DATA: %w", err)
	}

	return c.Quit()
}
