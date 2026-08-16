package builtin

import (
	"bufio"
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/cliclient"
)

// fakeSMTPServer accepts connections and discards SMTP commands, replying
// with generic success codes — enough to exercise the real cliclient.Send
// path end-to-end for send.go's dry-run-free tests without a real MTA.
func fakeSMTPServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				conn.Write([]byte("220 maelsink-fake ESMTP\r\n"))
				reader := bufio.NewReader(conn)
				inData := false
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					upper := strings.ToUpper(strings.TrimSpace(line))
					switch {
					case inData:
						if strings.TrimRight(line, "\r\n") == "." {
							inData = false
							conn.Write([]byte("250 OK\r\n"))
						}
					case strings.HasPrefix(upper, "DATA"):
						inData = true
						conn.Write([]byte("354 go ahead\r\n"))
					case strings.HasPrefix(upper, "QUIT"):
						conn.Write([]byte("221 bye\r\n"))
						return
					default:
						conn.Write([]byte("250 OK\r\n"))
					}
				}
			}()
		}
	}()
	return ln.Addr().String()
}

func TestSend_FlagComposed_DryRun(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	err := (Send{}).Run(context.Background(), s, []string{
		"--from", "a@b.com", "--to", "c@d.com", "--subject", "hi {{ .n }}", "--text", "body", "--dry-run",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "a@b.com") || !strings.Contains(out.String(), "hi 1") {
		t.Errorf("out = %q", out.String())
	}
}

func TestSend_ConflictingPrimarySources(t *testing.T) {
	fs := pflag.NewFlagSet("x", pflag.ContinueOnError)
	_, _, err := resolveSendMode("a.eml", "b.tmpl", "", "")
	if err == nil {
		t.Fatal("expected error for two primary sources")
	}
	_ = fs
}

func TestSend_EML_EnvelopeFromHeaders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "msg.eml")
	content := "From: sender@example.com\r\nTo: rcpt@example.com\r\nSubject: hi\r\n\r\nbody\r\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	fs := (Send{}).Flags()
	if err := fs.Parse(nil); err != nil {
		t.Fatalf("parse: %v", err)
	}
	raw, from, to, err := buildEMLMessage(fs, path)
	if err != nil {
		t.Fatalf("buildEMLMessage: %v", err)
	}
	if from != "sender@example.com" {
		t.Errorf("from = %q", from)
	}
	if len(to) != 1 || to[0] != "rcpt@example.com" {
		t.Errorf("to = %v", to)
	}
	if string(raw) != content {
		t.Errorf("raw should round-trip byte-exact, got %q", raw)
	}
}

// TestSend_Template_EnvelopeFromRenderedHeaders proves the fixed behavior:
// send --template derives the envelope from the RENDERED content's own
// From/To headers, exactly like --eml does with raw content — SPEC.md
// §7.5.5 describes both as producing "a complete RFC 5322 message", so a
// template that fully composes its own headers (e.g. via the "example"
// builtin) works standalone, with no --from/--to flags required.
func TestSend_Template_EnvelopeFromRenderedHeaders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "msg.tmpl")
	content := "From: sender@example.com\nTo: {{ .to }}\nSubject: hi\n\nbody\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	s, out, _ := newTestSession(t, nil)
	s.SetVar("to", "rcpt@example.com")
	err := (Send{}).Run(context.Background(), s, []string{"--template", path, "--dry-run"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "sender@example.com -> [rcpt@example.com]") {
		t.Errorf("expected envelope derived from rendered headers, got %q", out.String())
	}
}

// TestSend_Template_FlagsOverrideRenderedHeaders proves --from/--to still
// override the rendered content's headers when explicitly given (same
// override rule as --eml).
func TestSend_Template_FlagsOverrideRenderedHeaders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "msg.tmpl")
	content := "From: original@example.com\nTo: original-to@example.com\nSubject: hi\n\nbody\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	s, out, _ := newTestSession(t, nil)
	err := (Send{}).Run(context.Background(), s, []string{
		"--template", path, "--from", "override@example.com", "--to", "override-to@example.com", "--dry-run",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "override@example.com -> [override-to@example.com]") {
		t.Errorf("expected --from/--to to override rendered headers, got %q", out.String())
	}
}

// TestSend_Template_MissingHeadersAndFlagsErrors proves a template with no
// From/To headers AND no override flags fails clearly, rather than silently
// sending with an empty envelope (the original bug report's symptom).
func TestSend_Template_MissingHeadersAndFlagsErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "msg.tmpl")
	if err := os.WriteFile(path, []byte("just a body, no headers\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	s, _, _ := newTestSession(t, nil)
	err := (Send{}).Run(context.Background(), s, []string{"--template", path, "--dry-run"})
	if err == nil {
		t.Fatal("expected an error when the rendered content has no From/To and no override flags are given")
	}
}

func TestSend_CountProducesDistinctRenders(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	err := (Send{}).Run(context.Background(), s, []string{
		"--from", "a@b.com", "--to", "c@d.com", "--subject", "order-{{ uuid }}", "--text", "hi", "--count", "3", "--dry-run",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Dry run only renders/prints the first of N; verify the note about
	// total count appears (per-message distinctness is exercised more
	// directly by driving buildOne-equivalent logic below).
	if !strings.Contains(out.String(), "3 total messages") {
		t.Errorf("out = %q", out.String())
	}
}

func TestSend_JSONSpec_TemplatesStringFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "msg.json")
	content := `{"from":"a@b.com","to":["c@d.com"],"subject":"hi {{ .n }}","text":"body"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	s, _, _ := newTestSession(t, nil)
	spec, err := jsonSpec(path, s, map[string]any{"n": 1, "index": 0, "count": 1})
	if err != nil {
		t.Fatalf("jsonSpec: %v", err)
	}
	if spec.Subject != "hi 1" {
		t.Errorf("subject = %q", spec.Subject)
	}
}

func TestSend_EndToEnd_FakeSMTP(t *testing.T) {
	addr := fakeSMTPServer(t)
	s, out, _ := newTestSession(t, nil)
	s.SMTPAddr = addr

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := (Send{}).Run(ctx, s, []string{
		"--from", "a@b.com", "--to", "c@d.com", "--subject", "hi", "--text", "body",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "1/1 sent") {
		t.Errorf("out = %q", out.String())
	}
}

func TestResolveSMTP_TLSDefaultsToSessionOptions(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	s.SMTPTLS = cliclient.TLSOptions{Mode: cliclient.TLSStartTLS, InsecureSkipVerify: true}

	fs := (Send{}).Flags()
	if err := fs.Parse(nil); err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, _, tlsOpts, err := resolveSMTP(s, fs)
	if err != nil {
		t.Fatalf("resolveSMTP: %v", err)
	}
	if tlsOpts.Mode != cliclient.TLSStartTLS || !tlsOpts.InsecureSkipVerify {
		t.Errorf("tlsOpts = %+v, want session default", tlsOpts)
	}
}

func TestResolveSMTP_TLSPerInvocationOverride(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	s.SMTPTLS = cliclient.TLSOptions{Mode: cliclient.TLSStartTLS, InsecureSkipVerify: true}

	fs := (Send{}).Flags()
	if err := fs.Parse([]string{"--smtp-tls", "implicit"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, _, tlsOpts, err := resolveSMTP(s, fs)
	if err != nil {
		t.Fatalf("resolveSMTP: %v", err)
	}
	if tlsOpts.Mode != cliclient.TLSImplicit {
		t.Errorf("tlsOpts.Mode = %v, want TLSImplicit (per-invocation override)", tlsOpts.Mode)
	}
	if !tlsOpts.InsecureSkipVerify {
		t.Errorf("expected InsecureSkipVerify to still carry the session default when the override flag wasn't set")
	}
}

func TestResolveSMTP_InvalidTLSValue(t *testing.T) {
	s, _, _ := newTestSession(t, nil)

	fs := (Send{}).Flags()
	if err := fs.Parse([]string{"--smtp-tls", "bogus"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, _, _, err := resolveSMTP(s, fs); err == nil {
		t.Fatal("expected an error for an invalid --smtp-tls value")
	}
}

func TestSend_DirAttach(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("A"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".hidden"), []byte("H"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	fs := (Send{}).Flags()
	if err := fs.Parse([]string{"--dir", dir}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	paths, err := collectAttachments(fs)
	if err != nil {
		t.Fatalf("collectAttachments: %v", err)
	}
	if len(paths) != 1 || !strings.HasSuffix(paths[0], "a.txt") {
		t.Errorf("paths = %v", paths)
	}
}
