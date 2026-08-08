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

	results, err := s.SearchFTS(ctx, "invoice")
	if err != nil {
		t.Fatalf("SearchFTS: %v", err)
	}
	if len(results) != 1 || results[0].ID != m1.ID {
		t.Fatalf("expected 1 result matching m1, got %+v", results)
	}

	results, err = s.SearchFTS(ctx, "bob")
	if err != nil {
		t.Fatalf("SearchFTS: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected both messages to match on to_addrs, got %d", len(results))
	}
}
