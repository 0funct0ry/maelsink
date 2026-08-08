package store

import (
	"context"
	"sync"
	"testing"
)

func TestMemoryStore_SaveGetRoundtrip(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	msg := &Message{Subject: "hello"}
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if msg.ID == "" {
		t.Fatal("Save did not assign an ID")
	}

	got, err := s.Get(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Subject != "hello" {
		t.Fatalf("got subject %q, want %q", got.Subject, "hello")
	}
}

func TestMemoryStore_GetMissing(t *testing.T) {
	s := NewMemoryStore()
	if _, err := s.Get(context.Background(), "nope"); err != ErrNotFound {
		t.Fatalf("Get missing: got %v, want ErrNotFound", err)
	}
}

func TestMemoryStore_ListOrderingAndPagination(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	var ids []string
	for i := 0; i < 5; i++ {
		msg := &Message{}
		if err := s.Save(ctx, msg); err != nil {
			t.Fatalf("Save: %v", err)
		}
		ids = append(ids, msg.ID)
	}

	all, total, err := s.List(ctx, ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 5 {
		t.Fatalf("total = %d, want 5", total)
	}
	if len(all) != 5 {
		t.Fatalf("len(all) = %d, want 5", len(all))
	}
	// newest first: last saved id comes first
	for i, msg := range all {
		want := ids[len(ids)-1-i]
		if msg.ID != want {
			t.Fatalf("all[%d].ID = %q, want %q", i, msg.ID, want)
		}
	}

	page, total, err := s.List(ctx, ListFilter{Limit: 2, Offset: 1})
	if err != nil {
		t.Fatalf("List paginated: %v", err)
	}
	if total != 5 {
		t.Fatalf("paginated total = %d, want 5", total)
	}
	if len(page) != 2 {
		t.Fatalf("len(page) = %d, want 2", len(page))
	}
	if page[0].ID != ids[3] || page[1].ID != ids[2] {
		t.Fatalf("unexpected page contents: %+v", page)
	}
}

func TestMemoryStore_ListOffsetBeyondTotal(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	_ = s.Save(ctx, &Message{})

	page, total, err := s.List(ctx, ListFilter{Offset: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(page) != 0 {
		t.Fatalf("len(page) = %d, want 0", len(page))
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	msg := &Message{}
	_ = s.Save(ctx, msg)

	if err := s.Delete(ctx, msg.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, msg.ID); err != ErrNotFound {
		t.Fatalf("Get after delete: got %v, want ErrNotFound", err)
	}
	if err := s.Delete(ctx, msg.ID); err != ErrNotFound {
		t.Fatalf("Delete missing: got %v, want ErrNotFound", err)
	}
}

func TestMemoryStore_DeleteReindexesRemaining(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	a := &Message{}
	b := &Message{}
	c := &Message{}
	_ = s.Save(ctx, a)
	_ = s.Save(ctx, b)
	_ = s.Save(ctx, c)

	if err := s.Delete(ctx, a.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := s.Get(ctx, b.ID); err != nil {
		t.Fatalf("Get b after deleting a: %v", err)
	}
	if _, err := s.Get(ctx, c.ID); err != nil {
		t.Fatalf("Get c after deleting a: %v", err)
	}
}

func TestMemoryStore_Clear(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	_ = s.Save(ctx, &Message{})
	_ = s.Save(ctx, &Message{})

	if err := s.Clear(ctx); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	_, total, err := s.List(ctx, ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 0 {
		t.Fatalf("total after Clear = %d, want 0", total)
	}
}

func TestMemoryStore_ConcurrentSaveList(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_ = s.Save(ctx, &Message{})
			_, _, _ = s.List(ctx, ListFilter{Limit: 5})
		}()
	}
	wg.Wait()

	_, total, err := s.List(ctx, ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != n {
		t.Fatalf("total = %d, want %d", total, n)
	}
}
