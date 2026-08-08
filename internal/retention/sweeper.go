// Package retention implements the background sweeper that enforces
// storage.retention.max_messages / max_age_hours by deleting the oldest
// messages first, per SPEC.md §6. It deletes through the same
// store.MessageStore.Delete path as manual deletes, so future eventing
// (M7.0) applies uniformly to both.
package retention

import (
	"context"
	"log/slog"
	"time"

	"github.com/0funct0ry/maelsink/internal/store"
)

// Clock abstracts time.Now for deterministic tests.
type Clock interface {
	Now() time.Time
}

// RealClock is the production Clock.
type RealClock struct{}

// Now returns time.Now().
func (RealClock) Now() time.Time { return time.Now() }

// Config controls sweep behavior. A zero MaxMessages or MaxAgeHours means
// "unlimited" for that dimension.
type Config struct {
	MaxMessages int
	MaxAgeHours int
	Interval    time.Duration
}

// Sweeper periodically enforces Config against a store.MessageStore.
type Sweeper struct {
	store  store.MessageStore
	cfg    Config
	clock  Clock
	logger *slog.Logger
}

// New returns a ready-to-run Sweeper.
func New(s store.MessageStore, cfg Config, clock Clock, logger *slog.Logger) *Sweeper {
	if logger == nil {
		logger = slog.Default()
	}
	return &Sweeper{store: s, cfg: cfg, clock: clock, logger: logger}
}

// Run blocks, ticking at cfg.Interval and calling sweepOnce, until ctx is
// canceled.
func (sw *Sweeper) Run(ctx context.Context) {
	if sw.cfg.Interval <= 0 {
		<-ctx.Done()
		return
	}

	ticker := time.NewTicker(sw.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := sw.sweepOnce(ctx); err != nil {
				sw.logger.Error("retention sweep failed", "error", err)
			}
		}
	}
}

// sweepOnce runs a single enforcement pass: oldest-first deletion for any
// configured limit that is exceeded. Both limits at 0 means "unlimited" —
// nothing is deleted.
func (sw *Sweeper) sweepOnce(ctx context.Context) error {
	all, _, err := sw.store.List(ctx, store.ListFilter{})
	if err != nil {
		return err
	}

	// all is newest-first (per MessageStore.List's documented order);
	// reverse to get oldest-first for eviction ordering.
	oldest := make([]*store.Message, len(all))
	for i, m := range all {
		oldest[len(all)-1-i] = m
	}

	toDelete := map[string]struct{}{}

	if sw.cfg.MaxAgeHours > 0 {
		cutoff := sw.clock.Now().Add(-time.Duration(sw.cfg.MaxAgeHours) * time.Hour)
		for _, m := range oldest {
			if m.ReceivedAt.Before(cutoff) {
				toDelete[m.ID] = struct{}{}
			}
		}
	}

	if sw.cfg.MaxMessages > 0 && len(oldest) > sw.cfg.MaxMessages {
		excess := len(oldest) - sw.cfg.MaxMessages
		for _, m := range oldest[:excess] {
			toDelete[m.ID] = struct{}{}
		}
	}

	for _, m := range oldest {
		if _, ok := toDelete[m.ID]; !ok {
			continue
		}
		if err := sw.store.Delete(ctx, m.ID); err != nil && err != store.ErrNotFound {
			return err
		}
		sw.logger.Info("retention sweep deleted message", "msg_id", m.ID)
	}

	return nil
}
