package tmpl

import (
	"mime"
	"mime/quotedprintable"
	"strings"
	"text/template"
	"time"
)

// mimeFuncMap returns email/MIME encoding helper template functions.
func (e *Engine) mimeFuncMap() template.FuncMap {
	return template.FuncMap{
		"quotedPrintable": quotedPrintableEncode,
		"mimeWord":        mimeWordEncode,
		"rfc2822Date":     rfc2822Date,
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
