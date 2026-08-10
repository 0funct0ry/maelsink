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

func TestMemoryStore_GetByPrefix(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	msg := &Message{Subject: "hello"}
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.Get(ctx, msg.ID[:8])
	if err != nil {
		t.Fatalf("Get by prefix: %v", err)
	}
	if got.ID != msg.ID {
		t.Fatalf("got.ID = %q, want %q", got.ID, msg.ID)
	}
}

func TestMemoryStore_GetByAmbiguousPrefix(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	a := &Message{ID: "aaaa1111aaaa1111aaaa1111"}
	b := &Message{ID: "aaaa2222aaaa2222aaaa2222"}
	if err := s.Save(ctx, a); err != nil {
		t.Fatalf("Save a: %v", err)
	}
	if err := s.Save(ctx, b); err != nil {
		t.Fatalf("Save b: %v", err)
	}

	if _, err := s.Get(ctx, "aaaa"); err != ErrAmbiguousID {
		t.Fatalf("Get ambiguous prefix: got %v, want ErrAmbiguousID", err)
	}
}

func TestMemoryStore_DeleteByPrefix(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	msg := &Message{}
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := s.Delete(ctx, msg.ID[:8]); err != nil {
		t.Fatalf("Delete by prefix: %v", err)
	}
	if _, err := s.Get(ctx, msg.ID); err != ErrNotFound {
		t.Fatalf("Get after prefix delete: got %v, want ErrNotFound", err)
	}
}

func TestMemoryStore_DeleteByAmbiguousPrefix(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	a := &Message{ID: "bbbb1111bbbb1111bbbb1111"}
	b := &Message{ID: "bbbb2222bbbb2222bbbb2222"}
	_ = s.Save(ctx, a)
	_ = s.Save(ctx, b)

	if err := s.Delete(ctx, "bbbb"); err != ErrAmbiguousID {
		t.Fatalf("Delete ambiguous prefix: got %v, want ErrAmbiguousID", err)
	}
	// Neither message should have been touched.
	if _, err := s.Get(ctx, a.ID); err != nil {
		t.Fatalf("Get a after failed ambiguous delete: %v", err)
	}
	if _, err := s.Get(ctx, b.ID); err != nil {
		t.Fatalf("Get b after failed ambiguous delete: %v", err)
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

func TestMemoryStore_MarkRead(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	msg := &Message{Subject: "hello"}
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.Get(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Read {
		t.Fatal("new message should default to unread")
	}

	if err := s.MarkRead(ctx, msg.ID, true); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if err := s.MarkRead(ctx, msg.ID, true); err != nil {
		t.Fatalf("MarkRead (idempotent second call): %v", err)
	}

	got, err = s.Get(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Get after MarkRead: %v", err)
	}
	if !got.Read {
		t.Fatal("expected message to be marked read")
	}

	if err := s.MarkRead(ctx, msg.ID, false); err != nil {
		t.Fatalf("MarkRead(false): %v", err)
	}
	got, err = s.Get(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Get after MarkRead(false): %v", err)
	}
	if got.Read {
		t.Fatal("expected message to be marked unread")
	}
}

func TestMemoryStore_MarkReadMissing(t *testing.T) {
	s := NewMemoryStore()
	if err := s.MarkRead(context.Background(), "nope", true); err != ErrNotFound {
		t.Fatalf("MarkRead missing: got %v, want ErrNotFound", err)
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

func TestMemoryStore_TagFilterAndTagsAggregate(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	_ = s.Save(ctx, &Message{Subject: "a", Tags: []string{"smoke", "release"}})
	_ = s.Save(ctx, &Message{Subject: "b", Tags: []string{"smoke"}})
	_ = s.Save(ctx, &Message{Subject: "c"})

	msgs, total, err := s.List(ctx, ListFilter{Tag: "smoke"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	for _, m := range msgs {
		if !tagsContain(m.Tags, "smoke") {
			t.Fatalf("message %q missing expected tag", m.Subject)
		}
	}

	tags, err := s.Tags(ctx)
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}
	counts := map[string]int{}
	for _, tc := range tags {
		counts[tc.Tag] = tc.Count
	}
	if counts["smoke"] != 2 {
		t.Fatalf("smoke count = %d, want 2", counts["smoke"])
	}
	if counts["release"] != 1 {
		t.Fatalf("release count = %d, want 1", counts["release"])
	}

	_, total, err = s.List(ctx, ListFilter{Tags: []string{"smoke", "release"}})
	if err != nil {
		t.Fatalf("List (tags any): %v", err)
	}
	if total != 2 {
		t.Fatalf("tags any total = %d, want 2", total)
	}

	_, total, err = s.List(ctx, ListFilter{Tags: []string{"smoke", "release"}, TagMode: "all"})
	if err != nil {
		t.Fatalf("List (tags all): %v", err)
	}
	if total != 1 {
		t.Fatalf("tags all total = %d, want 1", total)
	}
}

func TestMemoryStore_CcBccFilters(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	_ = s.Save(ctx, &Message{Subject: "a", Cc: []Address{{Address: "carol@example.com"}}})
	_ = s.Save(ctx, &Message{Subject: "b", Bcc: []Address{{Address: "dave@example.com"}}})
	_ = s.Save(ctx, &Message{Subject: "c"})

	_, total, err := s.List(ctx, ListFilter{Cc: "carol"})
	if err != nil {
		t.Fatalf("List (cc): %v", err)
	}
	if total != 1 {
		t.Fatalf("cc total = %d, want 1", total)
	}

	_, total, err = s.List(ctx, ListFilter{Bcc: "dave"})
	if err != nil {
		t.Fatalf("List (bcc): %v", err)
	}
	if total != 1 {
		t.Fatalf("bcc total = %d, want 1", total)
	}
}

func TestMemoryStore_AddRemoveTag(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	msg := &Message{Subject: "hello", Tags: []string{"smoke"}}
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := s.AddTag(ctx, msg.ID, "release"); err != nil {
		t.Fatalf("AddTag: %v", err)
	}
	if err := s.AddTag(ctx, msg.ID, "release"); err != nil {
		t.Fatalf("AddTag (idempotent second call): %v", err)
	}

	got, err := s.Get(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !tagsContain(got.Tags, "smoke") || !tagsContain(got.Tags, "release") {
		t.Fatalf("Tags = %v, want smoke and release", got.Tags)
	}

	if err := s.RemoveTag(ctx, msg.ID, "smoke"); err != nil {
		t.Fatalf("RemoveTag: %v", err)
	}
	if err := s.RemoveTag(ctx, msg.ID, "smoke"); err != nil {
		t.Fatalf("RemoveTag (idempotent second call): %v", err)
	}
	if err := s.RemoveTag(ctx, msg.ID, "never-existed"); err != nil {
		t.Fatalf("RemoveTag on absent tag: %v", err)
	}

	got, err = s.Get(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Get after RemoveTag: %v", err)
	}
	if tagsContain(got.Tags, "smoke") {
		t.Fatal("expected smoke to be removed")
	}
	if !tagsContain(got.Tags, "release") {
		t.Fatal("expected release to remain")
	}
}

func TestMemoryStore_AddRemoveTagMissing(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	if err := s.AddTag(ctx, "nope", "smoke"); err != ErrNotFound {
		t.Fatalf("AddTag missing: got %v, want ErrNotFound", err)
	}
	if err := s.RemoveTag(ctx, "nope", "smoke"); err != ErrNotFound {
		t.Fatalf("RemoveTag missing: got %v, want ErrNotFound", err)
	}
}

func TestMemoryStore_AddRemoveTagInvalid(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	msg := &Message{Subject: "hello"}
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := s.AddTag(ctx, msg.ID, "   "); err != ErrInvalidTag {
		t.Fatalf("AddTag with blank tag: got %v, want ErrInvalidTag", err)
	}
	if err := s.RemoveTag(ctx, msg.ID, ""); err != ErrInvalidTag {
		t.Fatalf("RemoveTag with empty tag: got %v, want ErrInvalidTag", err)
	}
}

func TestMemoryStore_ConcurrentAddTagList(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	msg := &Message{Subject: "hello"}
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 25; i++ {
			_ = s.AddTag(ctx, msg.ID, "smoke")
			_ = s.RemoveTag(ctx, msg.ID, "smoke")
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 25; i++ {
			_, _, _ = s.List(ctx, ListFilter{})
		}
	}()
	wg.Wait()
}

func TestMemoryStore_ReadHasAttachmentsParseWarningFilters(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	_ = s.Save(ctx, &Message{Subject: "unread"})
	readMsg := &Message{Subject: "read", Read: true}
	_ = s.Save(ctx, readMsg)
	_ = s.Save(ctx, &Message{Subject: "with-attachment", Attachments: []Attachment{{ID: "a1"}}})
	_ = s.Save(ctx, &Message{Subject: "warn", ParseWarning: true})

	trueVal, falseVal := true, false

	_, total, _ := s.List(ctx, ListFilter{Read: &falseVal})
	if total != 3 {
		t.Fatalf("unread total = %d, want 3", total)
	}
	_, total, _ = s.List(ctx, ListFilter{Read: &trueVal})
	if total != 1 {
		t.Fatalf("read total = %d, want 1", total)
	}
	_, total, _ = s.List(ctx, ListFilter{HasAttachments: &trueVal})
	if total != 1 {
		t.Fatalf("has_attachments total = %d, want 1", total)
	}
	_, total, _ = s.List(ctx, ListFilter{ParseWarning: &trueVal})
	if total != 1 {
		t.Fatalf("parse_warning total = %d, want 1", total)
	}

	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.UnreadCount != 3 {
		t.Fatalf("UnreadCount = %d, want 3", stats.UnreadCount)
	}
	if stats.AttachmentCount != 1 {
		t.Fatalf("AttachmentCount = %d, want 1", stats.AttachmentCount)
	}
	if stats.ParseWarningCount != 1 {
		t.Fatalf("ParseWarningCount = %d, want 1", stats.ParseWarningCount)
	}
}
