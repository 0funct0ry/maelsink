package sqlite

import (
	"context"
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
