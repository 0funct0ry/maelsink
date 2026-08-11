package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/0funct0ry/maelsink/internal/store"
)

func sampleSession() *store.Session {
	return &store.Session{
		ClientIP:   "10.0.0.1",
		ClientHELO: "client.example.com",
		StartedAt:  time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		Status:     "completed",
		Transcript: []store.TranscriptLine{
			{Direction: 'S', Line: "220 maelsink.test ESMTP maelsink", Position: 0},
			{Direction: 'C', Line: "EHLO client.example.com", Position: 1},
			{Direction: 'S', Line: "250 maelsink.test Hello client.example.com", Position: 2},
		},
	}
}

func TestStore_CreateAndGetSession(t *testing.T) {
	s, _ := newTestStore(t, false)
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
	if got.ClientIP != sess.ClientIP || got.ClientHELO != sess.ClientHELO || got.Status != sess.Status {
		t.Fatalf("got = %+v, want matching fields to %+v", got, sess)
	}
	if !got.StartedAt.Equal(sess.StartedAt) {
		t.Fatalf("StartedAt = %v, want %v", got.StartedAt, sess.StartedAt)
	}
	if len(got.Transcript) != 3 {
		t.Fatalf("Transcript length = %d, want 3", len(got.Transcript))
	}
	for i, line := range got.Transcript {
		if line.Position != i {
			t.Fatalf("Transcript[%d].Position = %d, want %d", i, line.Position, i)
		}
	}
	if got.Transcript[1].Direction != 'C' || got.Transcript[1].Line != "EHLO client.example.com" {
		t.Fatalf("Transcript[1] = %+v, want the EHLO client line", got.Transcript[1])
	}
}

func TestStore_GetSessionMissing(t *testing.T) {
	s, _ := newTestStore(t, false)
	if _, err := s.GetSession(context.Background(), "nope"); err != store.ErrNotFound {
		t.Fatalf("GetSession missing: got %v, want ErrNotFound", err)
	}
}

func TestStore_GetSessionByPrefix(t *testing.T) {
	s, _ := newTestStore(t, false)
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

func TestStore_GetSessionByAmbiguousPrefix(t *testing.T) {
	s, _ := newTestStore(t, false)
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

	if _, err := s.GetSession(ctx, "aaaa"); err != store.ErrAmbiguousID {
		t.Fatalf("GetSession ambiguous prefix: got %v, want ErrAmbiguousID", err)
	}
}

func TestStore_ListSessionsOrderPaginationAndFilter(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	for i, ip := range []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"} {
		sess := sampleSession()
		sess.ClientIP = ip
		sess.StartedAt = time.Date(2026, 1, 1, 12, i, 0, 0, time.UTC)
		if i == 1 {
			sess.Status = "aborted"
		}
		if err := s.CreateSession(ctx, sess); err != nil {
			t.Fatalf("CreateSession %d: %v", i, err)
		}
	}

	list, total, err := s.ListSessions(ctx, store.SessionListFilter{})
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if list[0].ClientIP != "10.0.0.3" || list[2].ClientIP != "10.0.0.1" {
		t.Fatalf("unexpected order: %+v", list)
	}

	page, total, err := s.ListSessions(ctx, store.SessionListFilter{Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("ListSessions paginated: %v", err)
	}
	if total != 3 || len(page) != 1 || page[0].ClientIP != "10.0.0.2" {
		t.Fatalf("paginated result = %+v (total %d), want [10.0.0.2] (total 3)", page, total)
	}

	filtered, total, err := s.ListSessions(ctx, store.SessionListFilter{Status: "aborted"})
	if err != nil {
		t.Fatalf("ListSessions filtered: %v", err)
	}
	if total != 1 || filtered[0].ClientIP != "10.0.0.2" {
		t.Fatalf("filtered result = %+v (total %d), want just 10.0.0.2", filtered, total)
	}
}

func TestStore_DeleteSession(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	sess := sampleSession()
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := s.DeleteSession(ctx, sess.ID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if _, err := s.GetSession(ctx, sess.ID); err != store.ErrNotFound {
		t.Fatalf("GetSession after delete: got %v, want ErrNotFound", err)
	}
	if err := s.DeleteSession(ctx, sess.ID); err != store.ErrNotFound {
		t.Fatalf("DeleteSession missing: got %v, want ErrNotFound", err)
	}

	var lineCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM session_lines WHERE session_id = ?`, sess.ID).Scan(&lineCount); err != nil {
		t.Fatalf("counting session_lines: %v", err)
	}
	if lineCount != 0 {
		t.Fatalf("session_lines not cascade-deleted: %d rows remain", lineCount)
	}
}

func TestStore_MessageSessionIDRoundTrip(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	sess := sampleSession()
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	msg := sampleMessage()
	msg.SessionID = sess.ID
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.Get(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.SessionID != sess.ID {
		t.Fatalf("SessionID = %q, want %q", got.SessionID, sess.ID)
	}
}

// TestStore_DeleteMessageLinkedToSession is a regression test: a session's
// message_id (M8.4) is a foreign key into messages, so deleting a message
// that a session cross-links to must not violate that FK — it should
// instead clear the session's link and leave the session (and its
// transcript) intact.
func TestStore_DeleteMessageLinkedToSession(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	sess := sampleSession()
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	msg := sampleMessage()
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	sess.MessageID = &msg.ID
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession (linking message): %v", err)
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

// TestStore_ClearWithSessionLinkedMessages is a regression test for the
// reported bug: `delete --all` (Store.Clear) failed with a FOREIGN KEY
// constraint violation whenever any stored message had a session
// cross-linking to it via sessions.message_id.
func TestStore_ClearWithSessionLinkedMessages(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	sess := sampleSession()
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	msg := sampleMessage()
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	sess.MessageID = &msg.ID
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession (linking message): %v", err)
	}

	if err := s.Clear(ctx); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	if _, err := s.Get(ctx, msg.ID); err != store.ErrNotFound {
		t.Fatalf("Get after Clear: got %v, want ErrNotFound", err)
	}
	got, err := s.GetSession(ctx, sess.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.MessageID != nil {
		t.Fatalf("session.MessageID = %v, want nil after Clear", *got.MessageID)
	}
}

func TestStore_AppendSessionLine(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	sess := sampleSession()
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := s.AppendSessionLine(ctx, sess.ID, store.TranscriptLine{Direction: 'C', Line: "QUIT", Position: 3}); err != nil {
		t.Fatalf("AppendSessionLine: %v", err)
	}

	got, err := s.GetSession(ctx, sess.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if len(got.Transcript) != 4 {
		t.Fatalf("Transcript length = %d, want 4", len(got.Transcript))
	}
	last := got.Transcript[3]
	if last.Direction != 'C' || last.Line != "QUIT" || last.Position != 3 {
		t.Fatalf("last transcript entry = %+v, want the appended QUIT line", last)
	}
}

func TestStore_DeleteSessionByPrefixAndAmbiguous(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	sess := sampleSession()
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := s.DeleteSession(ctx, sess.ID[:8]); err != nil {
		t.Fatalf("DeleteSession by prefix: %v", err)
	}
	if _, err := s.GetSession(ctx, sess.ID); err != store.ErrNotFound {
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
	if err := s.DeleteSession(ctx, "aaaa"); err != store.ErrAmbiguousID {
		t.Fatalf("DeleteSession ambiguous prefix: got %v, want ErrAmbiguousID", err)
	}
}

func TestStore_DeleteSessionClearsMessageSessionID(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	sess := sampleSession()
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	msg := sampleMessage()
	msg.SessionID = sess.ID
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
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

func TestStore_ClearSessions(t *testing.T) {
	s, _ := newTestStore(t, false)
	ctx := context.Background()

	sess := sampleSession()
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	msg := sampleMessage()
	msg.SessionID = sess.ID
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.CreateSession(ctx, sampleSession()); err != nil {
		t.Fatalf("CreateSession 2: %v", err)
	}

	if err := s.ClearSessions(ctx); err != nil {
		t.Fatalf("ClearSessions: %v", err)
	}

	_, total, err := s.ListSessions(ctx, store.SessionListFilter{})
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
