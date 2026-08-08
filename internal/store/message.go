// Package store defines maelsink's storage-agnostic message model and the
// MessageStore interface (SPEC.md §2.2/§4). It has no dependency on the SMTP
// protocol layer or any HTTP/UI package, so future storage backends (M2.0's
// SQLite implementation) and readers (the REST API, the Web UI) can depend on
// it directly without pulling in /internal/smtp.
package store

import "time"

// Header is a single raw header line, preserved in transmission order with
// duplicates intact (e.g. multiple "Received:" lines) — a plain
// map[string][]string (as produced by net/textproto) would lose ordering.
type Header struct {
	Name  string
	Value string
}

// Address is a single RFC 5322 mailbox (display name optional).
type Address struct {
	Name    string
	Address string
}

// InlineImage is a MIME part referenced by Content-ID (typically via a
// "cid:" URL in an HTML body).
type InlineImage struct {
	ContentID   string
	Filename    string
	ContentType string
	Size        int64
	Data        []byte
}

// Attachment is a MIME part with a filename and no Content-ID reference.
type Attachment struct {
	Filename    string
	ContentType string
	Size        int64
	Data        []byte
}

// Message is a captured email, fully decoded from its raw SMTP DATA payload.
//
// A Message is always produced for every successfully DATA-accepted SMTP
// transaction, even when the payload fails to parse as valid RFC 5322/MIME —
// in that case ParseWarning is set and ParseError describes what went wrong,
// but the message (and its RawSource) is still captured, never dropped.
type Message struct {
	ID         string
	ReceivedAt time.Time
	Size       int64

	// Envelope values, exactly as received via MAIL FROM / RCPT TO,
	// independent of whatever the parsed headers say.
	EnvelopeFrom string
	EnvelopeTo   []string

	// Headers is the flat, ordered, duplicate-preserving header list as
	// transmitted. From/To/Cc/Subject below are convenience values decoded
	// from these headers (falling back to envelope values when absent).
	Headers []Header
	From    []Address
	To      []Address
	Cc      []Address
	// Bcc is derived by the SMTP layer as EnvelopeTo minus (To ∪ Cc), since
	// a Bcc header is never present in a transmitted message by definition.
	Bcc     []Address
	Subject string

	TextBody string
	HTMLBody string

	InlineImages []InlineImage
	Attachments  []Attachment

	// RawSource is the full raw RFC 5322 payload as received, kept
	// regardless of parse outcome (needed for .eml export and for
	// diagnosing ParseWarning messages).
	RawSource []byte

	ParseWarning bool
	ParseError   string
}
