package msgspec

import (
	"testing"
	"time"

	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

func TestParseJitterPercentage(t *testing.T) {
	d, err := ParseJitter("20%", time.Second)
	if err != nil {
		t.Fatalf("ParseJitter: %v", err)
	}
	if d != 200*time.Millisecond {
		t.Fatalf("ParseJitter(20%%, 1s) = %v, want 200ms", d)
	}
}

func TestParseJitterDuration(t *testing.T) {
	d, err := ParseJitter("50ms", time.Second)
	if err != nil {
		t.Fatalf("ParseJitter: %v", err)
	}
	if d != 50*time.Millisecond {
		t.Fatalf("ParseJitter(50ms) = %v, want 50ms", d)
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

	sched := &IntervalScheduler{Profile: "poisson", Mean: time.Second, Tmpl: engine}
	const n = 20000
	below := 0
	for i := 0; i < n; i++ {
		if sched.Next() < time.Second {
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
	sched := &IntervalScheduler{Profile: "steady", Mean: mean, Jitter: jitter, Tmpl: engine}
	for i := 0; i < 1000; i++ {
		d := sched.Next()
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
		sched := &IntervalScheduler{Profile: "poisson", Mean: time.Second, Tmpl: engine}
		out := make([]time.Duration, 10)
		for i := range out {
			out[i] = sched.Next()
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
