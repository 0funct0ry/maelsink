package cliclient

import (
	"bytes"
	"context"
	"net"
	"net/mail"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMessageSpec_Build_TextOnly(t *testing.T) {
	spec := MessageSpec{From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi", Text: "hello"}
	raw, err := spec.Build(time.Now())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	m, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if got := m.Header.Get("Subject"); got != "hi" {
		t.Errorf("Subject = %q", got)
	}
	body, _ := readAll(m.Body)
	if !strings.Contains(body, "hello") {
		t.Errorf("body = %q", body)
	}
}

func TestMessageSpec_Build_Tags(t *testing.T) {
	spec := MessageSpec{From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi", Text: "hello", Tags: []string{"smoke", "release"}}
	raw, err := spec.Build(time.Now())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	m, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	got := m.Header[textproto.CanonicalMIMEHeaderKey("X-Tag")]
	if len(got) != 2 || got[0] != "smoke" || got[1] != "release" {
		t.Fatalf("X-Tag headers = %v, want [smoke release]", got)
	}
}

func TestMessageSpec_Build_TextAndHTML(t *testing.T) {
	spec := MessageSpec{From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi", Text: "plain", HTML: "<b>html</b>"}
	raw, err := spec.Build(time.Now())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	s := string(raw)
	if !strings.Contains(s, "multipart/alternative") {
		t.Errorf("expected multipart/alternative, got:\n%s", s)
	}
	if !strings.Contains(s, "plain") || !strings.Contains(s, "<b>html</b>") {
		t.Errorf("expected both bodies present, got:\n%s", s)
	}
}

func TestMessageSpec_Build_WithAttachment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invoice.txt")
	if err := os.WriteFile(path, []byte("invoice contents"), 0o644); err != nil {
		t.Fatal(err)
	}

	spec := MessageSpec{
		From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi", Text: "hello",
		Attachments: []AttachmentSpec{{Path: path}},
	}
	raw, err := spec.Build(time.Now())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	s := string(raw)
	if !strings.Contains(s, "multipart/mixed") {
		t.Errorf("expected multipart/mixed, got:\n%s", s)
	}
	if !strings.Contains(s, `filename="invoice.txt"`) {
		t.Errorf("expected filename in Content-Disposition, got:\n%s", s)
	}
}

func TestMessageSpec_Envelope(t *testing.T) {
	spec := MessageSpec{From: "a@b.com", To: []string{"x@y.com"}, Cc: []string{"z@y.com"}, Bcc: []string{"w@y.com"}}
	from, to := spec.Envelope()
	if from != "a@b.com" {
		t.Errorf("from = %q", from)
	}
	want := []string{"x@y.com", "z@y.com", "w@y.com"}
	if len(to) != len(want) {
		t.Fatalf("to = %v, want %v", to, want)
	}
	for i := range want {
		if to[i] != want[i] {
			t.Errorf("to[%d] = %q, want %q", i, to[i], want[i])
		}
	}
}

// fakeSMTPServer accepts one SMTP transaction over a raw TCP listener,
// capturing MAIL FROM / RCPT TO / DATA without a real store — enough to
// verify cliclient.Send's wire behavior in isolation from internal/smtp.
func fakeSMTPServer(t *testing.T) (addr string, dataCh <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ch := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		tp := textproto.NewConn(conn)
		tp.PrintfLine("220 fake.local ESMTP")
		for {
			line, err := tp.ReadLine()
			if err != nil {
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
				tp.PrintfLine("250 fake.local")
			case strings.HasPrefix(line, "MAIL FROM"):
				tp.PrintfLine("250 OK")
			case strings.HasPrefix(line, "RCPT TO"):
				tp.PrintfLine("250 OK")
			case line == "DATA":
				tp.PrintfLine("354 go ahead")
				dr := tp.DotReader()
				data, _ := readAllBytes(dr)
				ch <- string(data)
				tp.PrintfLine("250 OK")
			case line == "QUIT":
				tp.PrintfLine("221 bye")
				return
			}
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return ln.Addr().String(), ch
}

func TestSend_Success(t *testing.T) {
	addr, dataCh := fakeSMTPServer(t)

	spec := MessageSpec{From: "a@b.com", To: []string{"c@d.com"}, Subject: "hi", Text: "hello"}
	raw, err := spec.Build(time.Now())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	from, to := spec.Envelope()

	if err := Send(context.Background(), addr, nil, from, to, raw); err != nil {
		t.Fatalf("Send: %v", err)
	}

	select {
	case data := <-dataCh:
		if !strings.Contains(data, "hello") {
			t.Errorf("server received unexpected data: %q", data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for DATA")
	}
}

func readAll(r interface{ Read([]byte) (int, error) }) (string, error) {
	var buf bytes.Buffer
	b := make([]byte, 4096)
	for {
		n, err := r.Read(b)
		buf.Write(b[:n])
		if err != nil {
			break
		}
	}
	return buf.String(), nil
}

func readAllBytes(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var buf bytes.Buffer
	b := make([]byte, 4096)
	for {
		n, err := r.Read(b)
		buf.Write(b[:n])
		if err != nil {
			break
		}
	}
	return buf.Bytes(), nil
}
