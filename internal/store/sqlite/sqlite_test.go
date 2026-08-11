package sqlite

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/0funct0ry/maelsink/internal/store"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func newTestStore(t *testing.T, attachOnDisk bool) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()

	db, err := Open(ctx, filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	diskPath := filepath.Join(dir, "attachments")
	return New(db, attachOnDisk, diskPath), diskPath
}

func sampleMessage() *store.Message {
	return &store.Message{
		ReceivedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		Size:       1234,
		Headers: []store.Header{
			{Name: "From", Value: "alice@example.com"},
			{Name: "Subject", Value: "hello"},
		},
		From:     []store.Address{{Address: "alice@example.com"}},
		To:       []store.Address{{Address: "bob@example.com"}},
		Cc:       []store.Address{{Address: "carol@example.com"}},
		Subject:  "hello",
		TextBody: "hi there",
		HTMLBody: "<p>hi there</p>",
		InlineImages: []store.InlineImage{
			{ContentID: "img1", Filename: "logo.png", ContentType: "image/png", Size: 3, Data: []byte("png")},
		},
		Attachments: []store.Attachment{
			{Filename: "doc.pdf", ContentType: "application/pdf", Size: 3, Data: []byte("pdf")},
		},
		RawSource: []byte("raw source bytes"),
	}
}

func TestOpen_WALModeEnabled(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	var mode string
	if err := db.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatalf("querying journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Fatalf("expected journal_mode=wal, got %q", mode)
	}
}

func TestStore_SaveGetRoundTrip_BlobAttachments(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	msg := sampleMessage()
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if msg.ID == "" {
		t.Fatal("expected Save to assign an ID")
	}

	got, err := s.Get(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Subject != "hello" || got.TextBody != "hi there" || got.HTMLBody != "<p>hi there</p>" {
		t.Fatalf("body fields mismatch: %+v", got)
	}
	if len(got.Headers) != 2 || got.Headers[0].Name != "From" || got.Headers[1].Name != "Subject" {
		t.Fatalf("headers order/content mismatch: %+v", got.Headers)
	}
	if len(got.Attachments) != 1 || string(got.Attachments[0].Data) != "pdf" || got.Attachments[0].ID == "" {
		t.Fatalf("attachment mismatch: %+v", got.Attachments)
	}
	if len(got.InlineImages) != 1 || string(got.InlineImages[0].Data) != "png" || got.InlineImages[0].ContentID != "img1" {
		t.Fatalf("inline image mismatch: %+v", got.InlineImages)
	}
	if !got.ReceivedAt.Equal(msg.ReceivedAt) {
		t.Fatalf("received_at mismatch: got %v want %v", got.ReceivedAt, msg.ReceivedAt)
	}
}

func TestStore_AttachmentsOnDisk(t *testing.T) {
	s, diskPath := newTestStore(t, true)
	ctx := context.Background()

	msg := sampleMessage()
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	entries, err := os.ReadDir(diskPath)
	if err != nil {
		t.Fatalf("reading disk path: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 files on disk (attachment + inline image), got %d", len(entries))
	}

	got, err := s.Get(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got.Attachments[0].Data) != "pdf" {
		t.Fatalf("expected attachment bytes read back from disk, got %q", got.Attachments[0].Data)
	}
	if string(got.InlineImages[0].Data) != "png" {
		t.Fatalf("expected inline image bytes read back from disk, got %q", got.InlineImages[0].Data)
	}

	var diskPathCol []string
	rows, err := s.db.QueryContext(ctx, `SELECT disk_path FROM attachments WHERE message_id = ?`, msg.ID)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			t.Fatalf("scan: %v", err)
		}
		diskPathCol = append(diskPathCol, p)
	}
	for _, p := range diskPathCol {
		if p == "" {
			t.Fatalf("expected non-empty disk_path when store_on_disk=true")
		}
	}
}

func TestStore_Delete(t *testing.T) {
	s, diskPath := newTestStore(t, true)
	ctx := context.Background()

	msg1 := sampleMessage()
	msg2 := sampleMessage()
	msg2.Subject = "second"
	if err := s.Save(ctx, msg1); err != nil {
		t.Fatalf("Save msg1: %v", err)
	}
	if err := s.Save(ctx, msg2); err != nil {
		t.Fatalf("Save msg2: %v", err)
	}

	if err := s.Delete(ctx, msg1.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := s.Get(ctx, msg1.ID); err != store.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}

	entries, err := os.ReadDir(diskPath)
	if err != nil {
		t.Fatalf("reading disk path: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected msg1's 2 attachment files removed, msg2's 2 remain, got %d entries", len(entries))
	}

	if err := s.Delete(ctx, msg1.ID); err != store.ErrNotFound {
		t.Fatalf("expected ErrNotFound deleting again, got %v", err)
	}
}

func TestStore_GetByPrefix(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	msg := sampleMessage()
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

func TestStore_GetByAmbiguousPrefix(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	msg1 := sampleMessage()
	msg1.ID = "aaaa1111aaaa1111aaaa1111"
	msg2 := sampleMessage()
	msg2.ID = "aaaa2222aaaa2222aaaa2222"
	if err := s.Save(ctx, msg1); err != nil {
		t.Fatalf("Save msg1: %v", err)
	}
	if err := s.Save(ctx, msg2); err != nil {
		t.Fatalf("Save msg2: %v", err)
	}

	if _, err := s.Get(ctx, "aaaa"); err != store.ErrAmbiguousID {
		t.Fatalf("Get ambiguous prefix: got %v, want ErrAmbiguousID", err)
	}
}

func TestStore_DeleteByPrefix(t *testing.T) {
	s, diskPath := newTestStore(t, true)
	ctx := context.Background()

	msg := sampleMessage()
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := s.Delete(ctx, msg.ID[:8]); err != nil {
		t.Fatalf("Delete by prefix: %v", err)
	}
	if _, err := s.Get(ctx, msg.ID); err != store.ErrNotFound {
		t.Fatalf("expected ErrNotFound after prefix delete, got %v", err)
	}

	entries, err := os.ReadDir(diskPath)
	if err != nil {
		t.Fatalf("reading disk path: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected attachment files removed by prefix delete, got %d entries", len(entries))
	}
}

func TestStore_DeleteByAmbiguousPrefix(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	msg1 := sampleMessage()
	msg1.ID = "bbbb1111bbbb1111bbbb1111"
	msg2 := sampleMessage()
	msg2.ID = "bbbb2222bbbb2222bbbb2222"
	_ = s.Save(ctx, msg1)
	_ = s.Save(ctx, msg2)

	if err := s.Delete(ctx, "bbbb"); err != store.ErrAmbiguousID {
		t.Fatalf("Delete ambiguous prefix: got %v, want ErrAmbiguousID", err)
	}
	if _, err := s.Get(ctx, msg1.ID); err != nil {
		t.Fatalf("Get msg1 after failed ambiguous delete: %v", err)
	}
	if _, err := s.Get(ctx, msg2.ID); err != nil {
		t.Fatalf("Get msg2 after failed ambiguous delete: %v", err)
	}
}

func TestStore_MarkRead(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	msg := sampleMessage()
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

func TestStore_MarkReadMissing(t *testing.T) {
	s, _ := newTestStore(t, false)
	if err := s.MarkRead(context.Background(), "nope", true); err != store.ErrNotFound {
		t.Fatalf("MarkRead missing: got %v, want ErrNotFound", err)
	}
}

func TestStore_Clear(t *testing.T) {
	s, diskPath := newTestStore(t, true)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := s.Save(ctx, sampleMessage()); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	if err := s.Clear(ctx); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	_, total, err := s.List(ctx, store.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 0 {
		t.Fatalf("expected 0 messages after Clear, got %d", total)
	}

	entries, err := os.ReadDir(diskPath)
	if err != nil {
		t.Fatalf("reading disk path: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected disk attachment dir empty after Clear, got %d entries", len(entries))
	}
}

// TestStore_DurabilityAcrossReopen approximates the "kill -9 and restart"
// durability check: close the DB handle (simulating process exit) and
// reopen a fresh Store against the same file, asserting the message is
// still retrievable — the on-disk file, not the process, is what persists.
func TestStore_DurabilityAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "durable.db")
	ctx := context.Background()

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	s := New(db, false, filepath.Join(dir, "attachments"))

	msg := sampleMessage()
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	db2, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer db2.Close()
	s2 := New(db2, false, filepath.Join(dir, "attachments"))

	got, err := s2.Get(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Get after reopen: %v", err)
	}
	if got.Subject != msg.Subject {
		t.Fatalf("expected message to survive reopen, got subject %q", got.Subject)
	}
}

func TestStore_SearchFTS(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	m1 := sampleMessage()
	m1.Subject = "invoice for march"
	m2 := sampleMessage()
	m2.Subject = "weekly newsletter"

	if err := s.Save(ctx, m1); err != nil {
		t.Fatalf("Save m1: %v", err)
	}
	if err := s.Save(ctx, m2); err != nil {
		t.Fatalf("Save m2: %v", err)
	}

	results, _, err := s.List(ctx, store.ListFilter{Query: "invoice"})
	if err != nil {
		t.Fatalf("List with query: %v", err)
	}
	if len(results) != 1 || results[0].ID != m1.ID {
		t.Fatalf("expected 1 result matching m1, got %+v", results)
	}

	results, _, err = s.List(ctx, store.ListFilter{Query: "bob"})
	if err != nil {
		t.Fatalf("List with query: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected both messages to match on to_addrs, got %d", len(results))
	}
}

// TestStore_SearchFTS_InvalidSyntax asserts that malformed FTS5 query
// syntax (unary "-" on a bare word, an unterminated quote, a bare hyphen
// inside an unquoted term, ...) is classified as store.ErrInvalidQuery
// rather than a generic error, so the API layer can respond with a
// friendly 400 instead of leaking the raw SQLite/FTS5 error text.
func TestStore_SearchFTS_InvalidSyntax(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	for _, q := range []string{
		"certificate -urgent", // unary "-" on a bare word: "no such column: urgent"
		"can't",               // unterminated quote from the bare apostrophe
		"multi-factor",        // bare hyphen inside an unquoted term
	} {
		t.Run(q, func(t *testing.T) {
			_, _, err := s.List(ctx, store.ListFilter{Query: q})
			if !errors.Is(err, store.ErrInvalidQuery) {
				t.Fatalf("List(%q) error = %v, want errors.Is(_, store.ErrInvalidQuery)", q, err)
			}
		})
	}
}

func TestStore_List_FiltersSortAndPagination(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		m := sampleMessage()
		m.Subject = "message"
		m.From = []store.Address{{Address: "alice@example.com"}}
		m.To = []store.Address{{Address: "bob@example.com"}}
		m.ReceivedAt = base.Add(time.Duration(i) * time.Hour)
		if i == 4 {
			m.Subject = "special report"
			m.From = []store.Address{{Address: "carol@other.com"}}
		}
		if err := s.Save(ctx, m); err != nil {
			t.Fatalf("Save %d: %v", i, err)
		}
	}

	t.Run("pagination", func(t *testing.T) {
		page1, total, err := s.List(ctx, store.ListFilter{Limit: 2, Offset: 0})
		if err != nil {
			t.Fatalf("List page1: %v", err)
		}
		if total != 5 {
			t.Fatalf("expected total 5, got %d", total)
		}
		if len(page1) != 2 {
			t.Fatalf("expected 2 results, got %d", len(page1))
		}
		page2, _, err := s.List(ctx, store.ListFilter{Limit: 2, Offset: 2})
		if err != nil {
			t.Fatalf("List page2: %v", err)
		}
		if len(page2) != 2 {
			t.Fatalf("expected 2 results, got %d", len(page2))
		}
		if page1[0].ID == page2[0].ID {
			t.Fatalf("expected different results across pages")
		}
	})

	t.Run("sort ascending", func(t *testing.T) {
		results, _, err := s.List(ctx, store.ListFilter{Sort: store.SortReceivedAtAsc})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(results) != 5 {
			t.Fatalf("expected 5 results, got %d", len(results))
		}
		if !results[0].ReceivedAt.Equal(base) {
			t.Fatalf("expected oldest first, got %v", results[0].ReceivedAt)
		}
	})

	t.Run("sort descending default", func(t *testing.T) {
		results, _, err := s.List(ctx, store.ListFilter{})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if !results[0].ReceivedAt.Equal(base.Add(4 * time.Hour)) {
			t.Fatalf("expected newest first, got %v", results[0].ReceivedAt)
		}
	})

	t.Run("filter by from", func(t *testing.T) {
		results, total, err := s.List(ctx, store.ListFilter{From: "carol"})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 1 || len(results) != 1 {
			t.Fatalf("expected 1 match, got %d", len(results))
		}
	})

	t.Run("filter by subject", func(t *testing.T) {
		results, _, err := s.List(ctx, store.ListFilter{Subject: "special"})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(results) != 1 || results[0].Subject != "special report" {
			t.Fatalf("expected special report match, got %+v", results)
		}
	})

	t.Run("filter by since/until", func(t *testing.T) {
		results, _, err := s.List(ctx, store.ListFilter{Since: base.Add(2 * time.Hour), Until: base.Add(3 * time.Hour)})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 matches in range, got %d", len(results))
		}
	})

	t.Run("attachment count populated", func(t *testing.T) {
		results, _, err := s.List(ctx, store.ListFilter{Limit: 1})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if results[0].AttachmentCount != 2 {
			t.Fatalf("expected attachment_count 2 (1 attachment + 1 inline image), got %d", results[0].AttachmentCount)
		}
	})
}

func TestStore_Stats(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	stats, err := s.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats on empty store: %v", err)
	}
	if stats.TotalMessages != 0 || stats.OldestReceivedAt != nil || stats.NewestReceivedAt != nil {
		t.Fatalf("expected empty stats, got %+v", stats)
	}

	m1 := sampleMessage()
	m1.ReceivedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m1.Size = 100
	m2 := sampleMessage()
	m2.ReceivedAt = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	m2.Size = 200

	if err := s.Save(ctx, m1); err != nil {
		t.Fatalf("Save m1: %v", err)
	}
	if err := s.Save(ctx, m2); err != nil {
		t.Fatalf("Save m2: %v", err)
	}

	stats, err = s.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.TotalMessages != 2 {
		t.Fatalf("expected 2 messages, got %d", stats.TotalMessages)
	}
	if stats.TotalSizeBytes != 300 {
		t.Fatalf("expected total size 300, got %d", stats.TotalSizeBytes)
	}
	if stats.OldestReceivedAt == nil || !stats.OldestReceivedAt.Equal(m1.ReceivedAt) {
		t.Fatalf("expected oldest %v, got %v", m1.ReceivedAt, stats.OldestReceivedAt)
	}
	if stats.NewestReceivedAt == nil || !stats.NewestReceivedAt.Equal(m2.ReceivedAt) {
		t.Fatalf("expected newest %v, got %v", m2.ReceivedAt, stats.NewestReceivedAt)
	}
}

func TestStore_Ping(t *testing.T) {
	s, _ := newTestStore(t, false)
	if err := s.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestStore_TagsRoundTripAndFilter(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	m1 := sampleMessage()
	m1.Tags = []string{"smoke", "release"}
	if err := s.Save(ctx, m1); err != nil {
		t.Fatalf("Save m1: %v", err)
	}

	m2 := sampleMessage()
	m2.Tags = []string{"smoke"}
	if err := s.Save(ctx, m2); err != nil {
		t.Fatalf("Save m2: %v", err)
	}

	m3 := sampleMessage()
	if err := s.Save(ctx, m3); err != nil {
		t.Fatalf("Save m3: %v", err)
	}

	got, err := s.Get(ctx, m1.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "smoke" || got.Tags[1] != "release" {
		t.Fatalf("Get tags = %v, want [smoke release]", got.Tags)
	}

	msgs, total, err := s.List(ctx, store.ListFilter{Tag: "smoke"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	for _, m := range msgs {
		if len(m.Tags) == 0 {
			t.Fatalf("listed message missing tags")
		}
	}

	tags, err := s.ListTagsWithStats(ctx)
	if err != nil {
		t.Fatalf("ListTagsWithStats: %v", err)
	}
	counts := map[string]int{}
	for _, tc := range tags {
		counts[tc.Name] = tc.Count
	}
	if counts["smoke"] != 2 {
		t.Fatalf("smoke count = %d, want 2", counts["smoke"])
	}
	if counts["release"] != 1 {
		t.Fatalf("release count = %d, want 1", counts["release"])
	}

	_, total, err = s.List(ctx, store.ListFilter{Tags: []string{"smoke", "release"}})
	if err != nil {
		t.Fatalf("List (tags any): %v", err)
	}
	if total != 2 {
		t.Fatalf("tags any total = %d, want 2", total)
	}

	_, total, err = s.List(ctx, store.ListFilter{Tags: []string{"smoke", "release"}, TagMode: "all"})
	if err != nil {
		t.Fatalf("List (tags all): %v", err)
	}
	if total != 1 {
		t.Fatalf("tags all total = %d, want 1", total)
	}
}

func TestStore_AddRemoveTag(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	msg := sampleMessage()
	msg.Tags = []string{"smoke"}
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
	if len(got.Tags) != 2 {
		t.Fatalf("Tags = %v, want 2 entries", got.Tags)
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
	if len(got.Tags) != 1 || got.Tags[0] != "release" {
		t.Fatalf("Tags = %v, want [release]", got.Tags)
	}
}

func TestStore_AddRemoveTagMissing(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	if err := s.AddTag(ctx, "nope", "smoke"); err != store.ErrNotFound {
		t.Fatalf("AddTag missing: got %v, want ErrNotFound", err)
	}
	if err := s.RemoveTag(ctx, "nope", "smoke"); err != store.ErrNotFound {
		t.Fatalf("RemoveTag missing: got %v, want ErrNotFound", err)
	}
}

func TestStore_AddRemoveTagInvalid(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	msg := sampleMessage()
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := s.AddTag(ctx, msg.ID, "   "); err != store.ErrInvalidTag {
		t.Fatalf("AddTag with blank tag: got %v, want ErrInvalidTag", err)
	}
	if err := s.RemoveTag(ctx, msg.ID, ""); err != store.ErrInvalidTag {
		t.Fatalf("RemoveTag with empty tag: got %v, want ErrInvalidTag", err)
	}
}

func TestStore_ReadHasAttachmentsParseWarningFilters(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	unread := sampleMessage()
	unread.Attachments = nil
	unread.InlineImages = nil
	if err := s.Save(ctx, unread); err != nil {
		t.Fatalf("Save unread: %v", err)
	}

	read := sampleMessage()
	read.Read = true
	read.Attachments = nil
	read.InlineImages = nil
	if err := s.Save(ctx, read); err != nil {
		t.Fatalf("Save read: %v", err)
	}

	withAttachment := sampleMessage()
	if err := s.Save(ctx, withAttachment); err != nil {
		t.Fatalf("Save withAttachment: %v", err)
	}

	warn := sampleMessage()
	warn.Attachments = nil
	warn.InlineImages = nil
	warn.ParseWarning = true
	if err := s.Save(ctx, warn); err != nil {
		t.Fatalf("Save warn: %v", err)
	}

	trueVal, falseVal := true, false

	if _, total, err := s.List(ctx, store.ListFilter{Read: &falseVal}); err != nil || total != 3 {
		t.Fatalf("unread filter: total=%d err=%v, want 3", total, err)
	}
	if _, total, err := s.List(ctx, store.ListFilter{Read: &trueVal}); err != nil || total != 1 {
		t.Fatalf("read filter: total=%d err=%v, want 1", total, err)
	}
	if _, total, err := s.List(ctx, store.ListFilter{HasAttachments: &trueVal}); err != nil || total != 1 {
		t.Fatalf("has_attachments filter: total=%d err=%v, want 1", total, err)
	}
	if _, total, err := s.List(ctx, store.ListFilter{ParseWarning: &trueVal}); err != nil || total != 1 {
		t.Fatalf("parse_warning filter: total=%d err=%v, want 1", total, err)
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

func TestStore_ListTagsWithStats_ZeroMessageTag(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	if err := s.CreateTag(ctx, "empty", "indigo"); err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	m := &store.Message{Subject: "a", Tags: []string{"smoke"}}
	if err := s.Save(ctx, m); err != nil {
		t.Fatalf("Save: %v", err)
	}

	tags, err := s.ListTagsWithStats(ctx)
	if err != nil {
		t.Fatalf("ListTagsWithStats: %v", err)
	}
	byName := map[string]store.TagStats{}
	for _, tg := range tags {
		byName[tg.Name] = tg
	}
	empty, ok := byName["empty"]
	if !ok || empty.Count != 0 || empty.LastUsed != nil {
		t.Fatalf("empty tag stats = %+v, want Count=0, LastUsed=nil, ok=%v", empty, ok)
	}
	smoke, ok := byName["smoke"]
	if !ok || smoke.Count != 1 || smoke.LastUsed == nil {
		t.Fatalf("smoke tag stats = %+v, want Count=1 with LastUsed set, ok=%v", smoke, ok)
	}
}

func TestStore_RenameTag(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	m := &store.Message{Subject: "a", Tags: []string{"old"}}
	if err := s.Save(ctx, m); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := s.RenameTag(ctx, "old", "new"); err != nil {
		t.Fatalf("RenameTag: %v", err)
	}
	got, err := s.Get(ctx, m.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "new" {
		t.Fatalf("Tags = %v, want [new]", got.Tags)
	}

	if err := s.RenameTag(ctx, "does-not-exist", "whatever"); err != store.ErrTagNotFound {
		t.Fatalf("RenameTag missing: got %v, want ErrTagNotFound", err)
	}
}

func TestStore_RenameTagMerge(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	mA := &store.Message{Subject: "a", Tags: []string{"foo"}}
	mB := &store.Message{Subject: "b", Tags: []string{"bar"}}
	if err := s.Save(ctx, mA); err != nil {
		t.Fatalf("Save mA: %v", err)
	}
	if err := s.Save(ctx, mB); err != nil {
		t.Fatalf("Save mB: %v", err)
	}

	if err := s.RenameTag(ctx, "foo", "bar"); err != nil {
		t.Fatalf("RenameTag (merge): %v", err)
	}

	gotA, err := s.Get(ctx, mA.ID)
	if err != nil {
		t.Fatalf("Get mA: %v", err)
	}
	if len(gotA.Tags) != 1 || gotA.Tags[0] != "bar" {
		t.Fatalf("mA Tags = %v, want [bar]", gotA.Tags)
	}

	tags, err := s.ListTagsWithStats(ctx)
	if err != nil {
		t.Fatalf("ListTagsWithStats: %v", err)
	}
	for _, tg := range tags {
		if tg.Name == "foo" {
			t.Fatal("expected 'foo' tag row to be gone after merge")
		}
	}
}

func TestStore_RecolorTag(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	if err := s.CreateTag(ctx, "x", "indigo"); err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	if err := s.RecolorTag(ctx, "x", "amber"); err != nil {
		t.Fatalf("RecolorTag: %v", err)
	}
	tags, err := s.ListTagsWithStats(ctx)
	if err != nil {
		t.Fatalf("ListTagsWithStats: %v", err)
	}
	if len(tags) != 1 || tags[0].Color != "amber" {
		t.Fatalf("tags = %+v, want single tag with color amber", tags)
	}

	if err := s.RecolorTag(ctx, "does-not-exist", "amber"); err != store.ErrTagNotFound {
		t.Fatalf("RecolorTag missing: got %v, want ErrTagNotFound", err)
	}
	if err := s.RecolorTag(ctx, "x", "not-a-color"); err != store.ErrInvalidTag {
		t.Fatalf("RecolorTag invalid color: got %v, want ErrInvalidTag", err)
	}
}

func TestStore_CreateTag(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	if err := s.CreateTag(ctx, "fresh", "cyan"); err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	if err := s.CreateTag(ctx, "fresh", "cyan"); err != store.ErrTagExists {
		t.Fatalf("CreateTag duplicate: got %v, want ErrTagExists", err)
	}
	if err := s.CreateTag(ctx, "other", "not-a-color"); err != store.ErrInvalidTag {
		t.Fatalf("CreateTag invalid color: got %v, want ErrInvalidTag", err)
	}
}

func TestStore_DeleteTag(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	m := &store.Message{Subject: "a", Tags: []string{"gone"}}
	if err := s.Save(ctx, m); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := s.DeleteTag(ctx, "gone"); err != nil {
		t.Fatalf("DeleteTag: %v", err)
	}
	got, err := s.Get(ctx, m.ID)
	if err != nil {
		t.Fatalf("Get after DeleteTag: %v", err)
	}
	if len(got.Tags) != 0 {
		t.Fatalf("expected message to still exist but be untagged, got Tags=%v", got.Tags)
	}

	if err := s.DeleteTag(ctx, "gone"); err != store.ErrTagNotFound {
		t.Fatalf("DeleteTag missing: got %v, want ErrTagNotFound", err)
	}
}

func TestStore_DeleteTagWithMessages(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	mA := &store.Message{Subject: "a", Tags: []string{"doomed"}}
	mB := &store.Message{Subject: "b"}
	if err := s.Save(ctx, mA); err != nil {
		t.Fatalf("Save mA: %v", err)
	}
	if err := s.Save(ctx, mB); err != nil {
		t.Fatalf("Save mB: %v", err)
	}

	if err := s.DeleteTagWithMessages(ctx, "doomed"); err != nil {
		t.Fatalf("DeleteTagWithMessages: %v", err)
	}
	if _, err := s.Get(ctx, mA.ID); err != store.ErrNotFound {
		t.Fatalf("mA should be deleted, got err %v", err)
	}
	if _, err := s.Get(ctx, mB.ID); err != nil {
		t.Fatalf("mB should remain: %v", err)
	}

	if err := s.DeleteTagWithMessages(ctx, "doomed"); err != store.ErrTagNotFound {
		t.Fatalf("DeleteTagWithMessages missing: got %v, want ErrTagNotFound", err)
	}
}

func TestStore_TagBackfillOnOpen(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	ctx := context.Background()

	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	s := New(db, false, "")
	if err := s.Save(ctx, &store.Message{Subject: "a", Tags: []string{"legacy"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Registered automatically by Save in the new schema, so re-opening
	// should be a no-op for this tag rather than erroring or duplicating.
	db.Close()

	db2, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer db2.Close()
	s2 := New(db2, false, "")
	tags, err := s2.ListTagsWithStats(ctx)
	if err != nil {
		t.Fatalf("ListTagsWithStats: %v", err)
	}
	found := false
	for _, tg := range tags {
		if tg.Name == "legacy" {
			found = true
			if tg.Count != 1 {
				t.Fatalf("legacy tag count = %d, want 1", tg.Count)
			}
		}
	}
	if !found {
		t.Fatal("expected 'legacy' tag to survive reopen")
	}
}
