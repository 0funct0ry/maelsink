package lineedit

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestIDCacheTTL(t *testing.T) {
	c := NewIDCache(50 * time.Millisecond)
	var calls atomic.Int32
	fetch := func(ctx context.Context) []string {
		calls.Add(1)
		return []string{"a", "b"}
	}

	got := c.Get(context.Background(), fetch)
	if calls.Load() != 1 {
		t.Fatalf("first call: expected 1 fetch, got %d", calls.Load())
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 ids, got %v", got)
	}

	// Within TTL: should reuse cache, not call fetch again.
	c.Get(context.Background(), fetch)
	if calls.Load() != 1 {
		t.Fatalf("within TTL: expected fetch count to stay 1, got %d", calls.Load())
	}

	// After TTL elapses: should refetch.
	time.Sleep(70 * time.Millisecond)
	c.Get(context.Background(), fetch)
	if calls.Load() != 2 {
		t.Fatalf("after TTL: expected 2 fetches, got %d", calls.Load())
	}
}

func TestIDCacheCachesEmptyResult(t *testing.T) {
	c := NewIDCache(50 * time.Millisecond)
	var calls atomic.Int32
	fetch := func(ctx context.Context) []string {
		calls.Add(1)
		return nil
	}

	c.Get(context.Background(), fetch)
	c.Get(context.Background(), fetch)
	if calls.Load() != 1 {
		t.Fatalf("expected offline/empty result to still be cached; got %d fetches", calls.Load())
	}
}
