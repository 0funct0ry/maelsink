package builtin

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
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

func TestParseJitterPercentage(t *testing.T) {
	d, err := parseJitter("20%", time.Second)
	if err != nil {
		t.Fatalf("parseJitter: %v", err)
	}
	if d != 200*time.Millisecond {
		t.Fatalf("parseJitter(20%%, 1s) = %v, want 200ms", d)
	}
}

func TestParseJitterDuration(t *testing.T) {
	d, err := parseJitter("50ms", time.Second)
	if err != nil {
		t.Fatalf("parseJitter: %v", err)
	}
	if d != 50*time.Millisecond {
		t.Fatalf("parseJitter(50ms) = %v, want 50ms", d)
	}
}

// TestIntervalSchedulerPoissonIsExponentialNotUniform samples many poisson
// intervals at a fixed mean and checks the fraction below the mean is close
// to 1-e^-1 (~0.632, the CDF of an exponential distribution evaluated at its
// own mean) rather than ~0.5 (what a uniform distribution would give),
// confirming the profile draws from an exponential, not uniform,
// distribution (SPEC.md §7.6.3's Definition of Done).
func TestIntervalSchedulerPoissonIsExponentialNotUniform(t *testing.T) {
	engine, err := tmpl.New(99, false)
	if err != nil {
		t.Fatalf("tmpl.New: %v", err)
	}
	defer engine.Close()

	sched := &intervalScheduler{profile: "poisson", mean: time.Second, tmpl: engine}
	const n = 20000
	below := 0
	for i := 0; i < n; i++ {
		if sched.next() < time.Second {
			below++
		}
	}
	frac := float64(below) / float64(n)
	if frac < 0.58 || frac > 0.68 {
		t.Fatalf("fraction below mean = %.3f, want ~0.632 (exponential CDF at mean)", frac)
	}
}

// TestIntervalSchedulerSteadyStaysWithinJitterBand confirms "steady"
// produces intervals uniformly within [mean-jitter, mean+jitter], per
// SPEC.md §7.6.3.
func TestIntervalSchedulerSteadyStaysWithinJitterBand(t *testing.T) {
	engine, err := tmpl.New(1, false)
	if err != nil {
		t.Fatalf("tmpl.New: %v", err)
	}
	defer engine.Close()

	mean := 500 * time.Millisecond
	jitter := 100 * time.Millisecond
	sched := &intervalScheduler{profile: "steady", mean: mean, jitter: jitter, tmpl: engine}
	for i := 0; i < 1000; i++ {
		d := sched.next()
		if d < mean-jitter || d > mean+jitter {
			t.Fatalf("interval %v outside jitter band [%v,%v]", d, mean-jitter, mean+jitter)
		}
	}
}

// TestIntervalSchedulerSeedReproducesSequence confirms --seed reproduces the
// exact same sequence of intervals across two separate scheduler instances,
// per SPEC.md §7.6.3.
func TestIntervalSchedulerSeedReproducesSequence(t *testing.T) {
	draw := func(seed int64) []time.Duration {
		engine, err := tmpl.New(seed, false)
		if err != nil {
			t.Fatalf("tmpl.New: %v", err)
		}
		defer engine.Close()
		sched := &intervalScheduler{profile: "poisson", mean: time.Second, tmpl: engine}
		out := make([]time.Duration, 10)
		for i := range out {
			out[i] = sched.next()
		}
		return out
	}

	a := draw(5)
	b := draw(5)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("interval %d differs across same-seed runs: %v vs %v", i, a[i], b[i])
		}
	}
}
