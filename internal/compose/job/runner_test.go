package job

import (
	"context"
	"testing"
	"time"
)

func TestRunRandMsgSendsAndReportsProgress(t *testing.T) {
	addr, count := fakeSMTPServer(t)
	target := Target{SMTPAddr: addr}

	var lastSent, lastFailed int
	run := RunRandMsg(target, RandMsgParams{Count: 3, Concurrency: 2})
	if err := run(context.Background(), func(sent, failed int) { lastSent, lastFailed = sent, failed }); err != nil {
		t.Fatalf("RunRandMsg: %v", err)
	}
	if lastSent != 3 || lastFailed != 0 {
		t.Fatalf("progress = sent=%d failed=%d, want sent=3 failed=0", lastSent, lastFailed)
	}
	if got := count(); got != 3 {
		t.Fatalf("server received %d messages, want 3", got)
	}
}

func TestRunWeirdMsgSendsOne(t *testing.T) {
	addr, count := fakeSMTPServer(t)
	target := Target{SMTPAddr: addr}

	run := RunWeirdMsg(target, WeirdMsgParams{Kind: "unicode"})
	var sent, failed int
	if err := run(context.Background(), func(s, f int) { sent, failed = s, f }); err != nil {
		t.Fatalf("RunWeirdMsg: %v", err)
	}
	if sent != 1 || failed != 0 {
		t.Fatalf("progress = sent=%d failed=%d, want sent=1 failed=0", sent, failed)
	}
	if got := count(); got != 1 {
		t.Fatalf("server received %d messages, want 1", got)
	}
}

func TestRunBlastSendsToManyRecipients(t *testing.T) {
	addr, count := fakeSMTPServer(t)
	target := Target{SMTPAddr: addr}

	run := RunBlast(target, BlastParams{Recipients: 5, Split: "mixed"})
	var sent int
	if err := run(context.Background(), func(s, f int) { sent = s }); err != nil {
		t.Fatalf("RunBlast: %v", err)
	}
	if sent != 1 {
		t.Fatalf("progress sent = %d, want 1 (one SMTP transaction)", sent)
	}
	if got := count(); got != 1 {
		t.Fatalf("server received %d messages, want 1", got)
	}
}

func TestRunIntMsgCancelStopsPromptly(t *testing.T) {
	addr, _ := fakeSMTPServer(t)
	target := Target{SMTPAddr: addr}

	ctx, cancel := context.WithCancel(context.Background())
	run := RunIntMsg(target, IntMsgParams{IntervalMS: 10})

	done := make(chan error, 1)
	go func() { done <- run(ctx, func(sent, failed int) {}) }()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RunIntMsg did not stop within 2s of cancel")
	}
}

func TestRunDelugeDelegatesToRandMsg(t *testing.T) {
	addr, count := fakeSMTPServer(t)
	target := Target{SMTPAddr: addr}

	run := RunDeluge(target, DelugeParams{Count: 4, Concurrency: 4})
	var sent int
	if err := run(context.Background(), func(s, f int) { sent = s }); err != nil {
		t.Fatalf("RunDeluge: %v", err)
	}
	if sent != 4 {
		t.Fatalf("progress sent = %d, want 4", sent)
	}
	if got := count(); got != 4 {
		t.Fatalf("server received %d messages, want 4", got)
	}
}
