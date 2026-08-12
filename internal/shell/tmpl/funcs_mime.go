package tmpl

import (
	"mime"
	"mime/quotedprintable"
	"strings"
	"time"
)

// emailDocs documents email-composition-focused template functions: MIME
// encoding helpers, attachment chaining, and fake email addresses.
func (e *Engine) emailDocs() []FuncDoc {
	return []FuncDoc{
		{Name: "quotedPrintable", Category: CategoryEmail, Args: "s", Returns: "string",
			Description: "Quoted-printable encodes s (RFC 2045).", Fn: quotedPrintableEncode},
		{Name: "mimeWord", Category: CategoryEmail, Args: "s", Returns: "string",
			Description: "RFC 2047 encoded-word (UTF-8 Q-encoding) for non-ASCII email header values.", Fn: mimeWordEncode},
		{Name: "rfc2822Date", Category: CategoryEmail, Args: "[time]", Returns: "string",
			Description: "Formats the given time (default now) as RFC 1123Z, for email Date headers.", Fn: rfc2822Date},
		{Name: "fEmail", Category: CategoryEmail, Args: "[domain]", Returns: "string",
			Description: "Random email address, optionally on the given domain.", Fn: e.fEmail},
		{Name: "fileOf", Category: CategoryEmail, Args: "path", Returns: "string",
			Description: "Validates path exists and returns it unchanged (passthrough for chaining into an email's attachments).", Fn: e.fileOf},
		{Name: "attach", Category: CategoryEmail, Args: "path...", Returns: "string",
			Description: `Joins multiple file paths with "::" for send --attach's email-attachment chaining convention.`, Fn: e.attach},
	}
}

// quotedPrintableEncode encodes s using quoted-printable (RFC 2045).
func quotedPrintableEncode(s string) (string, error) {
	var sb strings.Builder
	w := quotedprintable.NewWriter(&sb)
	if _, err := w.Write([]byte(s)); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// mimeWordEncode encodes s per RFC 2047 (encoded-word), using UTF-8
// Q-encoding.
func mimeWordEncode(s string) string {
	return mime.QEncoding.Encode("UTF-8", s)
}

// rfc2822Date formats the given time (or now, if none given) per
// RFC 1123Z, as used in email Date headers.
func rfc2822Date(t ...time.Time) string {
	when := time.Now()
	if len(t) > 0 {
		when = t[0]
	}
	return when.Format(time.RFC1123Z)
}
