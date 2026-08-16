package compose

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/0funct0ry/maelsink/internal/cliclient"
)

// fakeSMTPServer accepts one SMTP transaction over a raw TCP listener,
// mirroring internal/cliclient's own test helper of the same name (kept
// local since that one is unexported in a different package) — enough to
// verify sendHandler's wire behavior without a real internal/smtp server.
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
				data, _ := io.ReadAll(tp.DotReader())
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

// fakeSMTPServerReject accepts a connection and immediately rejects MAIL
// FROM, letting send_test.go exercise sendHandler's SMTP-error path (no
// swallowing, per SPEC.md §7.7.5).
func fakeSMTPServerReject(t *testing.T) (addr string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
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
				tp.PrintfLine("550 mailbox unavailable")
			case line == "QUIT":
				tp.PrintfLine("221 bye")
				return
			}
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return ln.Addr().String()
}

func TestSendHandler(t *testing.T) {
	client := newTestClient(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))

	t.Run("eml success", func(t *testing.T) {
		smtpAddr, dataCh := fakeSMTPServer(t)
		engine := New(client, testLogger(), TargetConfig{SMTPAddr: smtpAddr}, Config{})

		rec := postRequest(t, engine, "/compose-api/v1/send", renderRequest{
			Format:   "eml",
			Template: "From: {{ .from }}\r\nTo: rcpt@example.com\r\nSubject: hi\r\n\r\nbody",
			Vars:     map[string]string{"from": "sender@example.com"},
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
		}
		select {
		case data := <-dataCh:
			if !strings.Contains(data, "sender@example.com") {
				t.Fatalf("SMTP server received unexpected data: %q", data)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for SMTP DATA")
		}
	})

	t.Run("json success", func(t *testing.T) {
		smtpAddr, dataCh := fakeSMTPServer(t)
		engine := New(client, testLogger(), TargetConfig{SMTPAddr: smtpAddr}, Config{})

		rec := postRequest(t, engine, "/compose-api/v1/send", renderRequest{
			Format:   "json",
			Template: `{"from":"a@example.com","to":["b@example.com"],"subject":"{{ .subj }}","text":"hello"}`,
			Vars:     map[string]string{"subj": "Rendered"},
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
		}
		select {
		case data := <-dataCh:
			if !strings.Contains(data, "Rendered") {
				t.Fatalf("SMTP server received unexpected data: %q", data)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for SMTP DATA")
		}
	})

	t.Run("smtp error surfaces raw text", func(t *testing.T) {
		smtpAddr := fakeSMTPServerReject(t)
		engine := New(client, testLogger(), TargetConfig{SMTPAddr: smtpAddr}, Config{})

		rec := postRequest(t, engine, "/compose-api/v1/send", renderRequest{
			Format:   "eml",
			Template: "From: a@example.com\r\nTo: b@example.com\r\nSubject: hi\r\n\r\nbody",
		})
		if rec.Code != http.StatusBadGateway {
			t.Fatalf("status = %d, want 502 (body %s)", rec.Code, rec.Body.String())
		}
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		errObj, _ := body["error"].(map[string]any)
		msg, _ := errObj["message"].(string)
		if !strings.Contains(msg, "550") {
			t.Fatalf("error.message = %q, want it to contain the raw SMTP rejection", msg)
		}
	})

	t.Run("json with a generated attachment", func(t *testing.T) {
		smtpAddr, dataCh := fakeSMTPServer(t)
		engine := New(client, testLogger(), TargetConfig{SMTPAddr: smtpAddr}, Config{})

		rec := postRequest(t, engine, "/compose-api/v1/send", renderRequest{
			Format:   "json",
			Template: `{"from":"a@example.com","to":["b@example.com"],"subject":"report","html":"<p>see attached</p>","attachments":[{"path":"{{ fCSV }}","filename":"report.csv"}]}`,
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
		}
		select {
		case data := <-dataCh:
			if !strings.Contains(data, "multipart/mixed") {
				t.Fatalf("SMTP server received unexpected data (want a multipart/mixed envelope): %q", data)
			}
			if !strings.Contains(data, `filename="report.csv"`) {
				t.Fatalf("SMTP server received unexpected data (want the attachment part): %q", data)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for SMTP DATA")
		}
	})

	t.Run("eml with a generated attachment", func(t *testing.T) {
		smtpAddr, dataCh := fakeSMTPServer(t)
		engine := New(client, testLogger(), TargetConfig{SMTPAddr: smtpAddr}, Config{})

		rec := postRequest(t, engine, "/compose-api/v1/send", renderRequest{
			Format:   "eml",
			Template: "From: a@example.com\r\nTo: b@example.com\r\nSubject: report\r\n\r\nsee attached",
			Attachments: []cliclient.AttachmentSpec{
				{Path: "{{ fCSV }}", Filename: "report.csv"},
			},
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
		}
		select {
		case data := <-dataCh:
			if !strings.Contains(data, "multipart/mixed") {
				t.Fatalf("SMTP server received unexpected data (want a multipart/mixed envelope): %q", data)
			}
			if !strings.Contains(data, `filename="report.csv"`) {
				t.Fatalf("SMTP server received unexpected data (want the attachment part): %q", data)
			}
			if !strings.Contains(data, "see attached") {
				t.Fatalf("SMTP server received unexpected data (want the original body preserved as the first part): %q", data)
			}
			if !strings.Contains(data, "Subject: report") {
				t.Fatalf("SMTP server received unexpected data (want the original headers preserved): %q", data)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for SMTP DATA")
		}
	})

	t.Run("eml with no attachments is sent byte-identical to the rendered template", func(t *testing.T) {
		smtpAddr, dataCh := fakeSMTPServer(t)
		engine := New(client, testLogger(), TargetConfig{SMTPAddr: smtpAddr}, Config{})

		rec := postRequest(t, engine, "/compose-api/v1/send", renderRequest{
			Format:   "eml",
			Template: "From: a@example.com\r\nTo: b@example.com\r\nSubject: plain\r\n\r\nplain body",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
		}
		select {
		case data := <-dataCh:
			if strings.Contains(data, "multipart/mixed") {
				t.Fatalf("SMTP server received unexpected multipart wrapping with no attachments: %q", data)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for SMTP DATA")
		}
	})

	t.Run("render error short-circuits before dialing SMTP", func(t *testing.T) {
		engine := New(client, testLogger(), TargetConfig{SMTPAddr: "127.0.0.1:1"}, Config{})

		rec := postRequest(t, engine, "/compose-api/v1/send", renderRequest{
			Format:   "eml",
			Template: "{{ .foo",
		})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (body %s)", rec.Code, rec.Body.String())
		}
	})
}
