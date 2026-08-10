package retention

import (
	"context"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/0funct0ry/maelsink/internal/events"
	"github.com/0funct0ry/maelsink/internal/store"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time { return c.now }

func seedMessages(t *testing.T, s store.MessageStore, ages []time.Duration, now time.Time) []string {
	t.Helper()
	var ids []string
	for _, age := range ages {
		msg := &store.Message{ReceivedAt: now.Add(-age)}
		if err := s.Save(context.Background(), msg); err != nil {
			t.Fatalf("seeding message: %v", err)
		}
		ids = append(ids, msg.ID)
	}
	return ids
}

func TestSweepOnce_MaxAgeHours(t *testing.T) {
	s := store.NewMemoryStore()
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	// oldest to newest: 10h, 5h, 1h ago.
	ids := seedMessages(t, s, []time.Duration{10 * time.Hour, 5 * time.Hour, 1 * time.Hour}, now)

	bus := events.NewBus()
	sub, unsub := bus.Subscribe()
	defer unsub()

	sw := New(s, bus, Config{MaxAgeHours: 6}, fakeClock{now: now}, nil)
	if err := sw.sweepOnce(context.Background()); err != nil {
		t.Fatalf("sweepOnce: %v", err)
	}

	select {
	case ev := <-sub:
		if ev.Type != events.TypeMessageDeleted {
			t.Fatalf("got event type %q, want %q", ev.Type, events.TypeMessageDeleted)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message.deleted event")
	}

	if _, err := s.Get(context.Background(), ids[0]); err != store.ErrNotFound {
		t.Fatalf("expected oldest (10h) message deleted, got %v", err)
	}
	if _, err := s.Get(context.Background(), ids[1]); err != nil {
		t.Fatalf("expected 5h message to survive (newer than 6h cutoff), got %v", err)
	}
	if _, err := s.Get(context.Background(), ids[2]); err != nil {
		t.Fatalf("expected 1h message to survive, got %v", err)
	}
}

func TestSweepOnce_MaxMessages(t *testing.T) {
	s := store.NewMemoryStore()
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)

	ages := make([]time.Duration, 10)
	for i := range ages {
		ages[i] = time.Duration(10-i) * time.Hour // oldest first
	}
	ids := seedMessages(t, s, ages, now)

	sw := New(s, events.NewBus(), Config{MaxMessages: 3}, fakeClock{now: now}, nil)
	if err := sw.sweepOnce(context.Background()); err != nil {
		t.Fatalf("sweepOnce: %v", err)
	}

	_, total, err := s.List(context.Background(), store.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected 3 messages remaining, got %d", total)
	}

	for i := 0; i < 7; i++ {
		if _, err := s.Get(context.Background(), ids[i]); err != store.ErrNotFound {
			t.Fatalf("expected oldest message %d deleted, got %v", i, err)
		}
	}
	for i := 7; i < 10; i++ {
		if _, err := s.Get(context.Background(), ids[i]); err != nil {
			t.Fatalf("expected newest message %d to survive, got %v", i, err)
		}
	}
}

func TestSweepOnce_UnlimitedWhenZero(t *testing.T) {
	s := store.NewMemoryStore()
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	seedMessages(t, s, []time.Duration{1000 * time.Hour, 2000 * time.Hour}, now)

	sw := New(s, events.NewBus(), Config{}, fakeClock{now: now}, nil)
	if err := sw.sweepOnce(context.Background()); err != nil {
		t.Fatalf("sweepOnce: %v", err)
	}

	_, total, err := s.List(context.Background(), store.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected both messages to survive with limits at 0, got %d", total)
	}
}

func TestRun_StopsOnContextCancel(t *testing.T) {
	s := store.NewMemoryStore()
	sw := New(s, events.NewBus(), Config{Interval: time.Millisecond}, RealClock{}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		sw.Run(ctx)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return promptly after context cancellation")
	}
}
