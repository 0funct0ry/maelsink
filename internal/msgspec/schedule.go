package msgspec

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

// IntervalScheduler draws the next inter-message gap per SPEC.md §7.6.3's
// three profiles, from a seeded tmpl.Engine so a fixed --seed reproduces
// both message content and timing.
type IntervalScheduler struct {
	Profile string // steady|poisson|bursty
	Mean    time.Duration
	Jitter  time.Duration
	Tmpl    *tmpl.Engine
}

// Next returns the next inter-message gap.
func (sc *IntervalScheduler) Next() time.Duration {
	switch sc.Profile {
	case "poisson":
		d := time.Duration(sc.Tmpl.ExpFloat64() * float64(sc.Mean))
		if d < 0 {
			d = 0
		}
		return d
	default: // steady, and bursty's quiet-period gap
		if sc.Jitter <= 0 {
			return sc.Mean
		}
		delta := time.Duration(sc.Tmpl.Float64()*2*float64(sc.Jitter)) - sc.Jitter
		d := sc.Mean + delta
		if d < 0 {
			d = 0
		}
		return d
	}
}

// ParseJitter accepts either a bare duration string (e.g. "200ms") or a
// percentage string (e.g. "20%"), the latter resolved against mean per
// SPEC.md §7.6.3.
func ParseJitter(s string, mean time.Duration) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
		return 0, nil
	}
	if strings.HasSuffix(s, "%") {
		pctStr := strings.TrimSuffix(s, "%")
		pct, err := strconv.ParseFloat(pctStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid jitter percentage %q: %w", s, err)
		}
		return time.Duration(float64(mean) * pct / 100), nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid jitter %q: %w", s, err)
	}
	return d, nil
}
