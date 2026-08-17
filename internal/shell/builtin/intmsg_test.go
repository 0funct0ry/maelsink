package builtin

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestIntMsgCountBounded(t *testing.T) {
	s, out, errBuf := newTestSession(t, nil)
	s.SMTPAddr = fakeSMTPServer(t)
	err := (IntMsg{}).Run(context.Background(), s, []string{
		"--interval", "1ms", "--count", "5", "--stats-interval", "1h",
	})
	if err != nil {
		t.Fatalf("Run: %v, stderr=%s", err, errBuf.String())
	}
	if !strings.Contains(out.String(), "sent=5") {
		t.Fatalf("expected sent=5 in summary, got: %s", out.String())
	}
}

func TestIntMsgDurationBounded(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	s.SMTPAddr = fakeSMTPServer(t)
	err := (IntMsg{}).Run(context.Background(), s, []string{
		"--interval", "1ms", "--duration", "30ms", "--stats-interval", "1h",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "elapsed=") {
		t.Fatalf("expected a summary line, got: %s", out.String())
	}
}

func TestIntMsgCtxCancelStopsCleanly(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	s.SMTPAddr = fakeSMTPServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	err := (IntMsg{}).Run(ctx, s, []string{"--interval", "1ms", "--stats-interval", "1h"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "sent=") {
		t.Fatalf("expected a summary line after cancellation, got: %s", out.String())
	}
}

func TestIntMsgConflictingIntervalAndRate(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	if err := (IntMsg{}).Run(context.Background(), s, []string{"--interval", "1s", "--rate", "2"}); err == nil {
		t.Fatalf("expected error when both --interval and --rate are set")
	}
}

func TestIntMsgBackgroundAndStop(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	s.SMTPAddr = fakeSMTPServer(t)

	if err := (IntMsg{}).Run(context.Background(), s, []string{
		"--interval", "5ms", "--background",
	}); err != nil {
		t.Fatalf("Run(--background): %v", err)
	}

	startMsg := out.String()
	if !strings.Contains(startMsg, "started in background") {
		t.Fatalf("expected a background-start message, got: %s", startMsg)
	}
	id := strings.TrimSuffix(strings.TrimSpace(startMsg), "\n")
	idx := strings.LastIndex(id, " ")
	id = id[idx+1:]

	// Let it send a few messages before stopping.
	time.Sleep(50 * time.Millisecond)

	out.Reset()
	if err := (IntMsg{}).Run(context.Background(), s, []string{"--stop", id}); err != nil {
		t.Fatalf("Run(--stop %s): %v", id, err)
	}
	if !strings.Contains(out.String(), "sent=") {
		t.Fatalf("expected a summary after --stop, got: %s", out.String())
	}

	// Stopping the same id twice should now fail: the job was removed.
	if err := (IntMsg{}).Run(context.Background(), s, []string{"--stop", id}); err == nil {
		t.Fatalf("expected error stopping an already-removed job")
	}
}

func TestIntMsgListShowsRunningAndFinished(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	s.SMTPAddr = fakeSMTPServer(t)

	out.Reset()
	if err := (IntMsg{}).Run(context.Background(), s, []string{"--list"}); err != nil {
		t.Fatalf("Run(--list) with no jobs: %v", err)
	}
	if !strings.Contains(out.String(), "no background intmsg runs") {
		t.Fatalf("expected empty-list message, got: %s", out.String())
	}

	// One long-running job, one that finishes almost immediately.
	if err := (IntMsg{}).Run(context.Background(), s, []string{"-i", "50ms", "--background"}); err != nil {
		t.Fatalf("Run(--background) #1: %v", err)
	}
	if err := (IntMsg{}).Run(context.Background(), s, []string{"-i", "1ms", "-n", "2", "--background"}); err != nil {
		t.Fatalf("Run(--background) #2: %v", err)
	}

	time.Sleep(50 * time.Millisecond) // let #2 reach its count and finish

	out.Reset()
	if err := (IntMsg{}).Run(context.Background(), s, []string{"--list"}); err != nil {
		t.Fatalf("Run(--list): %v", err)
	}
	listing := out.String()
	if !strings.Contains(listing, "running") {
		t.Fatalf("expected a running job in the listing, got: %s", listing)
	}
	if !strings.Contains(listing, "finished") {
		t.Fatalf("expected a finished job in the listing, got: %s", listing)
	}

	// Cleanup: stop both so they don't leak into other tests via s.
	for _, id := range []string{"1", "2"} {
		_ = (IntMsg{}).Run(context.Background(), s, []string{"--stop", id})
	}
}

func TestIntMsgStopUnknownID(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	if err := (IntMsg{}).Run(context.Background(), s, []string{"--stop", "does-not-exist"}); err == nil {
		t.Fatalf("expected error for unknown --stop id")
	}
}

func TestIntMsgBackgroundFinishesOnItsOwnBeforeStop(t *testing.T) {
	s, out, _ := newTestSession(t, nil)
	s.SMTPAddr = fakeSMTPServer(t)

	if err := (IntMsg{}).Run(context.Background(), s, []string{
		"--interval", "1ms", "--count", "3", "--background",
	}); err != nil {
		t.Fatalf("Run(--background): %v", err)
	}
	startMsg := out.String()
	idx := strings.LastIndex(strings.TrimSpace(startMsg), " ")
	id := strings.TrimSpace(startMsg)[idx+1:]

	// Give the background run time to reach its own --count limit and
	// finish naturally, before we ever call --stop.
	time.Sleep(100 * time.Millisecond)

	out.Reset()
	if err := (IntMsg{}).Run(context.Background(), s, []string{"--stop", id}); err != nil {
		t.Fatalf("Run(--stop %s) on an already-finished job: %v", id, err)
	}
	if !strings.Contains(out.String(), "sent=3") {
		t.Fatalf("expected sent=3 from the naturally-finished run, got: %s", out.String())
	}
}

func TestIntMsgShortFlags(t *testing.T) {
	s, out, errBuf := newTestSession(t, nil)
	s.SMTPAddr = fakeSMTPServer(t)
	err := (IntMsg{}).Run(context.Background(), s, []string{
		"-i", "1ms", "-n", "3", "-I", "1h", "-t", "to@example.com", "-f", "from@example.com",
	})
	if err != nil {
		t.Fatalf("Run: %v, stderr=%s", err, errBuf.String())
	}
	if !strings.Contains(out.String(), "sent=3") {
		t.Fatalf("expected sent=3, got: %s", out.String())
	}
}

func TestIntMsgInvalidProfile(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	if err := (IntMsg{}).Run(context.Background(), s, []string{"--profile", "chaotic"}); err == nil {
		t.Fatalf("expected error for invalid --profile")
	}
}

// Jitter-parsing and interval-scheduler distribution tests moved to
// internal/msgspec/schedule_test.go (M13.3): that package now owns
// parseJitter/intervalScheduler, shared with compose's job kinds.
