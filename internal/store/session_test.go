package store

import (
	"context"
	"testing"
	"time"
)

func sampleSession() *Session {
	return &Session{
		ClientIP:   "10.0.0.1",
		ClientHELO: "client.example.com",
		StartedAt:  time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		Status:     "completed",
		Transcript: []TranscriptLine{
			{Direction: 'S', Line: "220 maelsink.test ESMTP maelsink", Position: 0},
			{Direction: 'C', Line: "EHLO client.example.com", Position: 1},
		},
	}
}

func TestMemoryStore_CreateAndGetSession(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	sess := sampleSession()
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if sess.ID == "" {
		t.Fatal("CreateSession did not mint an ID")
	}

	got, err := s.GetSession(ctx, sess.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.ClientIP != sess.ClientIP || got.Status != sess.Status {
		t.Fatalf("got = %+v, want matching ClientIP/Status", got)
	}
	if len(got.Transcript) != 2 {
		t.Fatalf("Transcript length = %d, want 2", len(got.Transcript))
	}
}

func TestMemoryStore_GetSessionMissing(t *testing.T) {
	s := NewMemoryStore()
	if _, err := s.GetSession(context.Background(), "nope"); err != ErrNotFound {
		t.Fatalf("GetSession missing: got %v, want ErrNotFound", err)
	}
}

func TestMemoryStore_GetSessionByPrefix(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	sess := sampleSession()
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	got, err := s.GetSession(ctx, sess.ID[:8])
	if err != nil {
		t.Fatalf("GetSession by prefix: %v", err)
	}
	if got.ID != sess.ID {
		t.Fatalf("got.ID = %q, want %q", got.ID, sess.ID)
	}
}

func TestMemoryStore_GetSessionByAmbiguousPrefix(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	a := sampleSession()
	a.ID = "aaaa1111aaaa1111aaaa1111"
	b := sampleSession()
	b.ID = "aaaa2222aaaa2222aaaa2222"
	if err := s.CreateSession(ctx, a); err != nil {
		t.Fatalf("CreateSession a: %v", err)
	}
	if err := s.CreateSession(ctx, b); err != nil {
		t.Fatalf("CreateSession b: %v", err)
	}

	if _, err := s.GetSession(ctx, "aaaa"); err != ErrAmbiguousID {
		t.Fatalf("GetSession ambiguous prefix: got %v, want ErrAmbiguousID", err)
	}
}

func TestMemoryStore_ListSessionsOrderAndPagination(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	for i, ip := range []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"} {
		sess := sampleSession()
		sess.ClientIP = ip
		sess.StartedAt = time.Date(2026, 1, 1, 12, i, 0, 0, time.UTC)
		if err := s.CreateSession(ctx, sess); err != nil {
			t.Fatalf("CreateSession %d: %v", i, err)
		}
	}

	list, total, err := s.ListSessions(ctx, SessionListFilter{})
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	// Newest-started first by default.
	if list[0].ClientIP != "10.0.0.3" || list[2].ClientIP != "10.0.0.1" {
		t.Fatalf("unexpected order: %+v", list)
	}

	page, total, err := s.ListSessions(ctx, SessionListFilter{Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("ListSessions paginated: %v", err)
	}
	if total != 3 || len(page) != 1 || page[0].ClientIP != "10.0.0.2" {
		t.Fatalf("paginated result = %+v (total %d), want [10.0.0.2] (total 3)", page, total)
	}
}

func TestMemoryStore_ListSessionsFilterByStatus(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	completed := sampleSession()
	completed.Status = "completed"
	if err := s.CreateSession(ctx, completed); err != nil {
		t.Fatalf("CreateSession completed: %v", err)
	}
	aborted := sampleSession()
	aborted.Status = "aborted"
	if err := s.CreateSession(ctx, aborted); err != nil {
		t.Fatalf("CreateSession aborted: %v", err)
	}

	list, total, err := s.ListSessions(ctx, SessionListFilter{Status: "aborted"})
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if total != 1 || list[0].ID != aborted.ID {
		t.Fatalf("filtered result = %+v (total %d), want just the aborted session", list, total)
	}
}

func TestMemoryStore_DeleteSession(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	sess := sampleSession()
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := s.DeleteSession(ctx, sess.ID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if _, err := s.GetSession(ctx, sess.ID); err != ErrNotFound {
		t.Fatalf("GetSession after delete: got %v, want ErrNotFound", err)
	}
	if err := s.DeleteSession(ctx, sess.ID); err != ErrNotFound {
		t.Fatalf("DeleteSession missing: got %v, want ErrNotFound", err)
	}
}

// TestMemoryStore_DeleteMessageLinkedToSession verifies deleting a message
// clears any session's dangling MessageID cross-link, mirroring the SQLite
// backend's FK-safety fix for the same scenario (M8.4).
func TestMemoryStore_DeleteMessageLinkedToSession(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	msg := &Message{Subject: "hello"}
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	sess := sampleSession()
	sess.MessageID = &msg.ID
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := s.Delete(ctx, msg.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := s.GetSession(ctx, sess.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.MessageID != nil {
		t.Fatalf("session.MessageID = %v, want nil after its message was deleted", *got.MessageID)
	}
}

// TestMemoryStore_ClearWithSessionLinkedMessages mirrors the SQLite
// backend's regression test for the reported `delete --all` bug.
func TestMemoryStore_ClearWithSessionLinkedMessages(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	msg := &Message{Subject: "hello"}
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	sess := sampleSession()
	sess.MessageID = &msg.ID
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := s.Clear(ctx); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	got, err := s.GetSession(ctx, sess.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.MessageID != nil {
		t.Fatalf("session.MessageID = %v, want nil after Clear", *got.MessageID)
	}
}

func TestMemoryStore_AppendSessionLine(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	sess := sampleSession()
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := s.AppendSessionLine(ctx, sess.ID, TranscriptLine{Direction: 'C', Line: "QUIT", Position: 2}); err != nil {
		t.Fatalf("AppendSessionLine: %v", err)
	}

	got, err := s.GetSession(ctx, sess.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if len(got.Transcript) != 3 {
		t.Fatalf("Transcript length = %d, want 3", len(got.Transcript))
	}
	last := got.Transcript[2]
	if last.Direction != 'C' || last.Line != "QUIT" || last.Position != 2 {
		t.Fatalf("last transcript entry = %+v, want the appended QUIT line", last)
	}
}

func TestMemoryStore_AppendSessionLineMissing(t *testing.T) {
	s := NewMemoryStore()
	if err := s.AppendSessionLine(context.Background(), "nope", TranscriptLine{Direction: 'C', Line: "x"}); err != ErrNotFound {
		t.Fatalf("AppendSessionLine missing: got %v, want ErrNotFound", err)
	}
}

func TestMemoryStore_DeleteSessionByPrefixAndAmbiguous(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	sess := sampleSession()
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := s.DeleteSession(ctx, sess.ID[:8]); err != nil {
		t.Fatalf("DeleteSession by prefix: %v", err)
	}
	if _, err := s.GetSession(ctx, sess.ID); err != ErrNotFound {
		t.Fatalf("GetSession after delete: got %v, want ErrNotFound", err)
	}

	a := sampleSession()
	a.ID = "aaaa1111aaaa1111aaaa1111"
	b := sampleSession()
	b.ID = "aaaa2222aaaa2222aaaa2222"
	if err := s.CreateSession(ctx, a); err != nil {
		t.Fatalf("CreateSession a: %v", err)
	}
	if err := s.CreateSession(ctx, b); err != nil {
		t.Fatalf("CreateSession b: %v", err)
	}
	if err := s.DeleteSession(ctx, "aaaa"); err != ErrAmbiguousID {
		t.Fatalf("DeleteSession ambiguous prefix: got %v, want ErrAmbiguousID", err)
	}
}

func TestMemoryStore_DeleteSessionClearsMessageSessionID(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	msg := &Message{Subject: "hello"}
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	sess := sampleSession()
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	msg.SessionID = sess.ID
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save (linking): %v", err)
	}

	if err := s.DeleteSession(ctx, sess.ID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}

	got, err := s.Get(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.SessionID != "" {
		t.Fatalf("message.SessionID = %q, want empty after its session was deleted", got.SessionID)
	}
}

func TestMemoryStore_ClearSessions(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	msg := &Message{Subject: "hello"}
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	sess := sampleSession()
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	msg.SessionID = sess.ID
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save (linking): %v", err)
	}
	if err := s.CreateSession(ctx, sampleSession()); err != nil {
		t.Fatalf("CreateSession 2: %v", err)
	}

	if err := s.ClearSessions(ctx); err != nil {
		t.Fatalf("ClearSessions: %v", err)
	}

	_, total, err := s.ListSessions(ctx, SessionListFilter{})
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if total != 0 {
		t.Fatalf("total = %d, want 0", total)
	}
	got, err := s.Get(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.SessionID != "" {
		t.Fatalf("message.SessionID = %q, want empty after ClearSessions", got.SessionID)
	}
}
