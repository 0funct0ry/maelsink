package builtin

import (
	"context"
	"strings"
	"testing"
)

func TestBlastMixedSplitDistribution(t *testing.T) {
	s, out, errBuf := newTestSession(t, nil)
	s.SMTPAddr = fakeSMTPServer(t)
	if err := (Blast{}).Run(context.Background(), s, []string{"--recipients", "9", "--split", "mixed"}); err != nil {
		t.Fatalf("Run: %v, stderr=%s", err, errBuf.String())
	}
	if !strings.Contains(out.String(), "to=3 cc=3 bcc=3") {
		t.Fatalf("expected an even 3/3/3 mixed split for 9 recipients, got: %s", out.String())
	}
}

func TestBlastInvalidSplit(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	s.SMTPAddr = fakeSMTPServer(t)
	if err := (Blast{}).Run(context.Background(), s, []string{"--split", "carrier-pigeon"}); err == nil {
		t.Fatalf("expected error for invalid --split")
	}
}

func TestBlastDryRun(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	if err := (Blast{}).Run(context.Background(), s, []string{"--recipients", "5", "--split", "cc", "--dry-run"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "dry run") {
		t.Fatalf("expected dry run output, got: %s", out.String())
	}
}
