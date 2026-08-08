package smtp

import (
	"strings"
	"testing"

	"github.com/0funct0ry/maelsink/internal/store"
)

func crlf(s string) []byte {
	return []byte(strings.ReplaceAll(s, "\n", "\r\n"))
}

func TestParse_PlainText(t *testing.T) {
	raw := crlf(`From: Alice <alice@example.com>
To: Bob <bob@example.com>
Subject: Hello
Content-Type: text/plain; charset=utf-8

Hi Bob, this is a plain text message.
`)
	msg := Parse(raw)

	if msg.ParseWarning {
		t.Fatalf("unexpected ParseWarning: %s", msg.ParseError)
	}
	if msg.Subject != "Hello" {
		t.Errorf("Subject = %q, want %q", msg.Subject, "Hello")
	}
	if len(msg.From) != 1 || msg.From[0].Address != "alice@example.com" {
		t.Errorf("From = %+v", msg.From)
	}
	if !strings.Contains(msg.TextBody, "plain text message") {
		t.Errorf("TextBody = %q", msg.TextBody)
	}
}

func TestParse_MultipartAlternative(t *testing.T) {
	raw := crlf(`From: a@example.com
To: b@example.com
Subject: Alt
Content-Type: multipart/alternative; boundary="B1"

--B1
Content-Type: text/plain

plain part
--B1
Content-Type: text/html

<p>html part</p>
--B1--
`)
	msg := Parse(raw)

	if msg.ParseWarning {
		t.Fatalf("unexpected ParseWarning: %s", msg.ParseError)
	}
	if !strings.Contains(msg.TextBody, "plain part") {
		t.Errorf("TextBody = %q", msg.TextBody)
	}
	if !strings.Contains(msg.HTMLBody, "html part") {
		t.Errorf("HTMLBody = %q", msg.HTMLBody)
	}
}

func TestParse_MultipartMixedWithAttachment(t *testing.T) {
	raw := crlf(`From: a@example.com
To: b@example.com
Subject: Mixed
Content-Type: multipart/mixed; boundary="B2"

--B2
Content-Type: text/plain

body text
--B2
Content-Type: application/octet-stream; name="file.bin"
Content-Disposition: attachment; filename="file.bin"
Content-Transfer-Encoding: base64

aGVsbG8=
--B2--
`)
	msg := Parse(raw)

	if msg.ParseWarning {
		t.Fatalf("unexpected ParseWarning: %s", msg.ParseError)
	}
	if !strings.Contains(msg.TextBody, "body text") {
		t.Errorf("TextBody = %q", msg.TextBody)
	}
	if len(msg.Attachments) != 1 {
		t.Fatalf("Attachments = %+v, want 1", msg.Attachments)
	}
	att := msg.Attachments[0]
	if att.Filename != "file.bin" {
		t.Errorf("Filename = %q, want file.bin", att.Filename)
	}
	if string(att.Data) != "hello" {
		t.Errorf("Data = %q, want %q", att.Data, "hello")
	}
	if att.ID == "" {
		t.Errorf("Attachment.ID = %q, want non-empty", att.ID)
	}
}

func TestParse_MultipartRelatedInlineImage(t *testing.T) {
	raw := crlf(`From: a@example.com
To: b@example.com
Subject: Related
Content-Type: multipart/related; boundary="B3"

--B3
Content-Type: multipart/alternative; boundary="B3A"

--B3A
Content-Type: text/plain

see image
--B3A
Content-Type: text/html

<img src="cid:img1">
--B3A--
--B3
Content-Type: image/png
Content-ID: <img1>
Content-Transfer-Encoding: base64

aGVsbG8=
--B3--
`)
	msg := Parse(raw)

	if msg.ParseWarning {
		t.Fatalf("unexpected ParseWarning: %s", msg.ParseError)
	}
	if !strings.Contains(msg.HTMLBody, "cid:img1") {
		t.Errorf("HTMLBody = %q", msg.HTMLBody)
	}
	if len(msg.InlineImages) != 1 {
		t.Fatalf("InlineImages = %+v, want 1", msg.InlineImages)
	}
	if msg.InlineImages[0].ContentID != "img1" {
		t.Errorf("ContentID = %q, want img1", msg.InlineImages[0].ContentID)
	}
	if string(msg.InlineImages[0].Data) != "hello" {
		t.Errorf("Data = %q, want %q", msg.InlineImages[0].Data, "hello")
	}
	if msg.InlineImages[0].ID == "" {
		t.Errorf("InlineImage.ID = %q, want non-empty", msg.InlineImages[0].ID)
	}
}

func TestParse_MalformedHeadersSurvives(t *testing.T) {
	raw := []byte("this is not a valid header block at all\r\n\r\nbody text here")
	msg := Parse(raw)

	if msg == nil {
		t.Fatal("Parse returned nil")
	}
	if !strings.Contains(msg.TextBody, "body text here") {
		t.Errorf("TextBody = %q, want salvaged body", msg.TextBody)
	}
}

func TestParse_MalformedMultipartBoundarySurvives(t *testing.T) {
	raw := crlf(`From: a@example.com
To: b@example.com
Content-Type: multipart/mixed; boundary="B4"

this does not respect the boundary at all
and never terminates properly
`)
	msg := Parse(raw)

	if msg == nil {
		t.Fatal("Parse returned nil")
	}
	if !msg.ParseWarning {
		t.Error("expected ParseWarning for malformed multipart body")
	}
}

func TestParse_MissingBoundaryParameter(t *testing.T) {
	raw := crlf(`From: a@example.com
To: b@example.com
Content-Type: multipart/mixed

some body
`)
	msg := Parse(raw)

	if msg == nil {
		t.Fatal("Parse returned nil")
	}
	if !msg.ParseWarning {
		t.Error("expected ParseWarning for missing boundary parameter")
	}
}

func TestParse_RFC2047Subject(t *testing.T) {
	raw := crlf(`From: a@example.com
To: b@example.com
Subject: =?UTF-8?B?SGVsbG8gV29ybGQ=?=
Content-Type: text/plain

body
`)
	msg := Parse(raw)

	if msg.Subject != "Hello World" {
		t.Errorf("Subject = %q, want %q", msg.Subject, "Hello World")
	}
}

func TestParse_DuplicateHeadersPreservedInOrder(t *testing.T) {
	raw := crlf(`From: a@example.com
To: b@example.com
Received: first
Received: second
Content-Type: text/plain

body
`)
	msg := Parse(raw)

	var received []string
	for _, h := range msg.Headers {
		if strings.EqualFold(h.Name, "Received") {
			received = append(received, h.Value)
		}
	}
	if len(received) != 2 || received[0] != "first" || received[1] != "second" {
		t.Errorf("Received headers = %+v, want [first second]", received)
	}
}

func TestParse_NeverPanics(t *testing.T) {
	inputs := [][]byte{
		nil,
		{},
		[]byte("\x00\x01\x02garbage"),
		[]byte("Content-Type: multipart/mixed; boundary=\r\n\r\n--\r\n"),
	}
	for _, in := range inputs {
		msg := Parse(in)
		if msg == nil {
			t.Errorf("Parse(%q) returned nil", in)
		}
	}
}

func TestDeriveBcc(t *testing.T) {
	to := []store.Address{{Address: "to@example.com"}}
	cc := []store.Address{{Address: "cc@example.com"}}
	bcc := deriveBcc(to, cc, []string{"to@example.com", "cc@example.com", "hidden@example.com"})

	if len(bcc) != 1 || bcc[0].Address != "hidden@example.com" {
		t.Errorf("deriveBcc = %+v, want [hidden@example.com]", bcc)
	}
}
