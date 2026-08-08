package smtp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"

	"github.com/0funct0ry/maelsink/internal/store"
)

// Parse decodes a raw RFC 5322 message, as received via SMTP DATA, into a
// store.Message. It never returns nil and never panics: any failure at any
// stage — malformed headers, a broken MIME boundary, an undecodable part —
// is captured as ParseWarning/ParseError on a best-effort Message rather
// than losing the message (SPEC.md §4).
func Parse(raw []byte) (msg *store.Message) {
	msg = &store.Message{RawSource: raw, Size: int64(len(raw))}
	defer func() {
		if r := recover(); r != nil {
			msg.ParseWarning = true
			msg.ParseError = fmt.Sprintf("panic while parsing message: %v", r)
		}
	}()

	msg.Headers = parseRawHeaders(raw)

	m, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		msg.ParseWarning = true
		msg.ParseError = err.Error()
		salvageBody(msg, raw)
		return msg
	}

	msg.Subject = decodeHeaderValue(m.Header.Get("Subject"))
	msg.From = parseAddressList(m.Header.Get("From"))
	msg.To = parseAddressList(m.Header.Get("To"))
	msg.Cc = parseAddressList(m.Header.Get("Cc"))

	mediaType, params, err := mime.ParseMediaType(m.Header.Get("Content-Type"))
	if err != nil {
		// No usable Content-Type: treat the entire body as plain text.
		body, _ := io.ReadAll(m.Body)
		msg.TextBody = string(decodeTransferEncoding(body, m.Header.Get("Content-Transfer-Encoding")))
		return msg
	}

	if !strings.HasPrefix(mediaType, "multipart/") {
		body, _ := io.ReadAll(m.Body)
		assignSinglePartBody(msg, mediaType, decodeTransferEncoding(body, m.Header.Get("Content-Transfer-Encoding")))
		return msg
	}

	boundary := params["boundary"]
	if boundary == "" {
		msg.ParseWarning = true
		msg.ParseError = "multipart message missing boundary parameter"
		salvageBody(msg, raw)
		return msg
	}

	if err := walkMultipart(msg, m.Body, boundary); err != nil {
		msg.ParseWarning = true
		if msg.ParseError == "" {
			msg.ParseError = err.Error()
		}
	}
	return msg
}

// parseRawHeaders manually scans the header block (everything before the
// first blank line), preserving transmission order and duplicate headers —
// net/textproto.MIMEHeader (a map[string][]string) cannot represent order,
// which SPEC.md's "flat header list, ordered, duplicates preserved"
// requirement needs.
func parseRawHeaders(raw []byte) []store.Header {
	headerBlock := raw
	if idx := bytes.Index(raw, []byte("\r\n\r\n")); idx >= 0 {
		headerBlock = raw[:idx]
	} else if idx := bytes.Index(raw, []byte("\n\n")); idx >= 0 {
		headerBlock = raw[:idx]
	}

	lines := strings.Split(strings.ReplaceAll(string(headerBlock), "\r\n", "\n"), "\n")

	var headers []store.Header
	var curName, curValue string
	flush := func() {
		if curName != "" {
			headers = append(headers, store.Header{Name: curName, Value: strings.TrimSpace(curValue)})
		}
	}

	for _, line := range lines {
		if line == "" {
			continue
		}
		if (line[0] == ' ' || line[0] == '\t') && curName != "" {
			// Folded continuation line.
			curValue += " " + strings.TrimSpace(line)
			continue
		}
		flush()
		idx := strings.IndexByte(line, ':')
		if idx < 0 {
			curName, curValue = "", ""
			continue
		}
		curName = line[:idx]
		curValue = line[idx+1:]
	}
	flush()
	return headers
}

// decodeHeaderValue decodes RFC 2047 encoded-words (e.g. in Subject), best
// effort — an undecodable value is returned unchanged rather than dropped.
func decodeHeaderValue(v string) string {
	dec := new(mime.WordDecoder)
	decoded, err := dec.DecodeHeader(v)
	if err != nil {
		return v
	}
	return decoded
}

// parseAddressList decodes a From/To/Cc header value into store.Address
// values, returning nil (not an error) when the value is empty or
// unparseable — the caller falls back to envelope values in that case.
func parseAddressList(v string) []store.Address {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	addrs, err := mail.ParseAddressList(v)
	if err != nil {
		return nil
	}
	out := make([]store.Address, len(addrs))
	for i, a := range addrs {
		out[i] = store.Address{Name: a.Name, Address: a.Address}
	}
	return out
}

// salvageBody is the fallback used when a message can't be parsed as valid
// RFC 5322/MIME at all: everything after the first blank line (or the whole
// payload, if no blank line is found) becomes the text body, guaranteeing
// the message is still captured rather than dropped.
func salvageBody(msg *store.Message, raw []byte) {
	body := raw
	if idx := bytes.Index(raw, []byte("\r\n\r\n")); idx >= 0 {
		body = raw[idx+4:]
	} else if idx := bytes.Index(raw, []byte("\n\n")); idx >= 0 {
		body = raw[idx+2:]
	}
	msg.TextBody = string(body)
}

// decodeTransferEncoding decodes a MIME part body per its
// Content-Transfer-Encoding. On decode failure, the original bytes are
// returned rather than discarding content.
func decodeTransferEncoding(body []byte, cte string) []byte {
	switch strings.ToLower(strings.TrimSpace(cte)) {
	case "base64":
		decoded := make([]byte, base64.StdEncoding.DecodedLen(len(body)))
		n, err := base64.StdEncoding.Decode(decoded, bytes.TrimSpace(body))
		if err != nil {
			return body
		}
		return decoded[:n]
	case "quoted-printable":
		decoded, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body)))
		if err != nil {
			return body
		}
		return decoded
	default:
		return body
	}
}

// assignSinglePartBody places a non-multipart message's decoded body into
// TextBody or HTMLBody depending on its media type.
func assignSinglePartBody(msg *store.Message, mediaType string, body []byte) {
	if mediaType == "text/html" {
		msg.HTMLBody = string(body)
		return
	}
	msg.TextBody = string(body)
}

// walkMultipart recurses through a multipart body (handling nested
// multipart/alternative inside multipart/related/mixed, etc.), assigning
// each leaf part to TextBody, HTMLBody, InlineImages, or Attachments. A
// decode failure on one part is recorded via ParseWarning but does not
// abort processing of the remaining parts.
func walkMultipart(msg *store.Message, r io.Reader, boundary string) error {
	mr := multipart.NewReader(r, boundary)
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := processPart(msg, part); err != nil {
			msg.ParseWarning = true
			if msg.ParseError == "" {
				msg.ParseError = err.Error()
			}
		}
	}
}

func processPart(msg *store.Message, part *multipart.Part) error {
	defer part.Close()

	mediaType, params, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
	if err != nil {
		mediaType = "text/plain"
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return fmt.Errorf("nested multipart part missing boundary parameter")
		}
		return walkMultipart(msg, part, boundary)
	}

	body, err := io.ReadAll(part)
	if err != nil {
		return err
	}
	body = decodeTransferEncoding(body, part.Header.Get("Content-Transfer-Encoding"))

	contentID := strings.Trim(part.Header.Get("Content-Id"), "<>")
	filename := part.FileName()
	disposition := strings.ToLower(part.Header.Get("Content-Disposition"))

	switch {
	case contentID != "":
		msg.InlineImages = append(msg.InlineImages, store.InlineImage{
			ContentID:   contentID,
			Filename:    filename,
			ContentType: mediaType,
			Size:        int64(len(body)),
			Data:        body,
		})
	case filename != "" || strings.Contains(disposition, "attachment"):
		msg.Attachments = append(msg.Attachments, store.Attachment{
			Filename:    filename,
			ContentType: mediaType,
			Size:        int64(len(body)),
			Data:        body,
		})
	case mediaType == "text/html":
		msg.HTMLBody += string(body)
	case mediaType == "text/plain":
		msg.TextBody += string(body)
	default:
		// An unrecognized inline part with no filename/disposition: keep it
		// as an attachment rather than silently dropping content.
		msg.Attachments = append(msg.Attachments, store.Attachment{
			Filename:    filename,
			ContentType: mediaType,
			Size:        int64(len(body)),
			Data:        body,
		})
	}
	return nil
}

// deriveBcc computes Bcc = envelopeTo - (To ∪ Cc) by address, since a Bcc
// header is never present in a transmitted message by definition — the only
// way to recover it is by diffing the SMTP envelope against the parsed
// header recipients.
func deriveBcc(to, cc []store.Address, envelopeTo []string) []store.Address {
	seen := make(map[string]struct{}, len(to)+len(cc))
	for _, a := range to {
		seen[strings.ToLower(a.Address)] = struct{}{}
	}
	for _, a := range cc {
		seen[strings.ToLower(a.Address)] = struct{}{}
	}

	var bcc []store.Address
	for _, addr := range envelopeTo {
		if _, ok := seen[strings.ToLower(addr)]; !ok {
			bcc = append(bcc, store.Address{Address: addr})
		}
	}
	return bcc
}
