package builtin

import (
	"context"
	"strings"
	"testing"
)

func TestWeirdMsgKinds(t *testing.T) {
	for _, kind := range append([]string{"random"}, weirdKinds...) {
		if kind == "thread" {
			continue // exercised separately: sends multiple messages
		}
		t.Run(kind, func(t *testing.T) {
			s, out, errBuf := newTestSession(t, nil)
			s.SMTPAddr = fakeSMTPServer(t)
			if err := (WeirdMsg{}).Run(context.Background(), s, []string{"--kind", kind}); err != nil {
				t.Fatalf("Run(%s): %v, stderr=%s", kind, err, errBuf.String())
			}
			if !strings.Contains(out.String(), "sent ") {
				t.Fatalf("kind %s: expected sent confirmation, got: %s", kind, out.String())
			}
		})
	}
}

func TestWeirdMsgThread(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	s.SMTPAddr = fakeSMTPServer(t)
	if err := (WeirdMsg{}).Run(context.Background(), s, []string{"--kind", "thread", "--depth", "3"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "sent 3 thread messages") {
		t.Fatalf("expected 3 thread messages sent, got: %s", out.String())
	}
}

func TestWeirdMsgInvalidKind(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	s.SMTPAddr = fakeSMTPServer(t)
	if err := (WeirdMsg{}).Run(context.Background(), s, []string{"--kind", "nonsense"}); err == nil {
		t.Fatalf("expected error for invalid --kind")
	}
}

func TestWeirdMsgHugeRespectsSize(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	s.SMTPAddr = fakeSMTPServer(t)
	if err := (WeirdMsg{}).Run(context.Background(), s, []string{"--kind", "huge", "--size", "1KB"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
}
