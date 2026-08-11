package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/0funct0ry/maelsink/internal/events"
	"github.com/0funct0ry/maelsink/internal/store"
)

func sampleSession(clientIP, status string, startedAt time.Time) *store.Session {
	return &store.Session{
		ClientIP:   clientIP,
		ClientHELO: "client.example.com",
		StartedAt:  startedAt,
		Status:     status,
		Transcript: []store.TranscriptLine{
			{Direction: 'S', Line: "220 maelsink.test ESMTP maelsink", Position: 0},
			{Direction: 'C', Line: "EHLO client.example.com", Position: 1},
		},
	}
}

func TestAPI_ListSessions(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	for i, ip := range []string{"10.0.0.1", "10.0.0.2"} {
		sess := sampleSession(ip, "completed", time.Now().Add(time.Duration(i)*time.Minute))
		if err := s.CreateSession(ctx, sess); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
	}

	router := newRouter(t, s, Config{})
	rec, body := doJSON(t, router, http.MethodGet, "/api/v1/sessions", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %v)", rec.Code, body)
	}

	sessions, ok := body["sessions"].([]any)
	if !ok || len(sessions) != 2 {
		t.Fatalf("sessions = %v, want 2 entries", body["sessions"])
	}
	if body["total"].(float64) != 2 {
		t.Fatalf("total = %v, want 2", body["total"])
	}
	first := sessions[0].(map[string]any)
	if first["client_ip"] != "10.0.0.2" {
		t.Fatalf("first session client_ip = %v, want newest-started first (10.0.0.2)", first["client_ip"])
	}
	if _, hasTranscript := first["transcript"]; hasTranscript {
		t.Fatal("list response should not include a transcript field")
	}
}

func TestAPI_GetSession(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	msg := sampleMessage("hi", "a@example.com", "b@example.com", time.Now())
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	sess := sampleSession("10.0.0.1", "completed", time.Now())
	sess.MessageID = &msg.ID
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	router := newRouter(t, s, Config{})
	rec, body := doJSON(t, router, http.MethodGet, "/api/v1/sessions/"+sess.ID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %v)", rec.Code, body)
	}
	if body["id"] != sess.ID {
		t.Fatalf("id = %v, want %q", body["id"], sess.ID)
	}
	if body["message_id"] != msg.ID {
		t.Fatalf("message_id = %v, want %q", body["message_id"], msg.ID)
	}
	transcript, ok := body["transcript"].([]any)
	if !ok || len(transcript) != 2 {
		t.Fatalf("transcript = %v, want 2 entries", body["transcript"])
	}
	line0 := transcript[0].(map[string]any)
	if line0["direction"] != "S" || line0["position"].(float64) != 0 {
		t.Fatalf("transcript[0] = %v, want direction S position 0", line0)
	}
}

func TestAPI_GetSessionByPrefix(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	sess := sampleSession("10.0.0.1", "completed", time.Now())
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	router := newRouter(t, s, Config{})
	rec, body := doJSON(t, router, http.MethodGet, "/api/v1/sessions/"+sess.ID[:8], nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %v)", rec.Code, body)
	}
	if body["id"] != sess.ID {
		t.Fatalf("id = %v, want %q", body["id"], sess.ID)
	}
}

func TestAPI_GetSessionNotFound(t *testing.T) {
	s, _ := newTestStore(t)
	router := newRouter(t, s, Config{})

	rec, body := doJSON(t, router, http.MethodGet, "/api/v1/sessions/000000000000000000000000", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body: %v)", rec.Code, body)
	}
	errObj := body["error"].(map[string]any)
	if errObj["code"] != "session_not_found" {
		t.Fatalf("error.code = %v, want session_not_found", errObj["code"])
	}
}

func TestAPI_GetSessionAmbiguousPrefix(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	a := sampleSession("10.0.0.1", "completed", time.Now())
	a.ID = "aaaa1111aaaa1111aaaa1111"
	b := sampleSession("10.0.0.2", "completed", time.Now())
	b.ID = "aaaa2222aaaa2222aaaa2222"
	if err := s.CreateSession(ctx, a); err != nil {
		t.Fatalf("CreateSession a: %v", err)
	}
	if err := s.CreateSession(ctx, b); err != nil {
		t.Fatalf("CreateSession b: %v", err)
	}

	router := newRouter(t, s, Config{})
	rec, body := doJSON(t, router, http.MethodGet, "/api/v1/sessions/aaaa", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body: %v)", rec.Code, body)
	}
	errObj := body["error"].(map[string]any)
	if errObj["code"] != "ambiguous_id" {
		t.Fatalf("error.code = %v, want ambiguous_id", errObj["code"])
	}
}

func TestAPI_ListSessionsFilterByStatus(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	if err := s.CreateSession(ctx, sampleSession("10.0.0.1", "completed", time.Now())); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := s.CreateSession(ctx, sampleSession("10.0.0.2", "aborted", time.Now())); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	router := newRouter(t, s, Config{})
	rec, body := doJSON(t, router, http.MethodGet, "/api/v1/sessions?status=aborted", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %v)", rec.Code, body)
	}
	sessions := body["sessions"].([]any)
	if len(sessions) != 1 {
		t.Fatalf("sessions = %v, want 1 entry", sessions)
	}
	first := sessions[0].(map[string]any)
	if first["client_ip"] != "10.0.0.2" {
		t.Fatalf("filtered session client_ip = %v, want 10.0.0.2", first["client_ip"])
	}
}

// TestAPI_MessageDetailCarriesSessionID verifies the Message Detail ->
// Session Detail cross-link field is present when the message was produced
// by a tracked session.
func TestAPI_MessageDetailCarriesSessionID(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	sess := sampleSession("10.0.0.1", "completed", time.Now())
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	msg := sampleMessage("hi", "a@example.com", "b@example.com", time.Now())
	msg.SessionID = sess.ID
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	router := newRouter(t, s, Config{})
	rec, body := doJSON(t, router, http.MethodGet, "/api/v1/messages/"+msg.ID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %v)", rec.Code, body)
	}
	if body["session_id"] != sess.ID {
		t.Fatalf("session_id = %v, want %q", body["session_id"], sess.ID)
	}
}

func TestAPI_DeleteSession(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	sess := sampleSession("10.0.0.1", "completed", time.Now())
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	router, bus := newRouterWithBus(t, s, Config{})
	sub, unsub := bus.Subscribe()
	defer unsub()

	rec, body := doJSON(t, router, http.MethodDelete, "/api/v1/sessions/"+sess.ID, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %v)", rec.Code, body)
	}
	if _, err := s.GetSession(ctx, sess.ID); err != store.ErrNotFound {
		t.Fatalf("expected session deleted, got %v", err)
	}

	select {
	case ev := <-sub:
		if ev.Type != events.TypeSessionDeleted {
			t.Fatalf("got event type %q, want %q", ev.Type, events.TypeSessionDeleted)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for session.deleted event")
	}
}

func TestAPI_DeleteSessionByPrefix(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	sess := sampleSession("10.0.0.1", "completed", time.Now())
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	router := newRouter(t, s, Config{})
	rec, body := doJSON(t, router, http.MethodDelete, "/api/v1/sessions/"+sess.ID[:8], nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %v)", rec.Code, body)
	}
}

func TestAPI_DeleteSessionNotFound(t *testing.T) {
	s, _ := newTestStore(t)
	router := newRouter(t, s, Config{})

	rec, body := doJSON(t, router, http.MethodDelete, "/api/v1/sessions/000000000000000000000000", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body: %v)", rec.Code, body)
	}
}

// TestAPI_DeleteSessionClearsMessageSessionID is a regression test
// mirroring the message-deletion fix: deleting a session that a message
// cross-links to (via message.session_id) must not fail with a foreign
// key violation, and must clear that message's session_id rather than
// deleting the message.
func TestAPI_DeleteSessionClearsMessageSessionID(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	sess := sampleSession("10.0.0.1", "completed", time.Now())
	if err := s.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	msg := sampleMessage("hi", "a@example.com", "b@example.com", time.Now())
	msg.SessionID = sess.ID
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	router := newRouter(t, s, Config{})
	rec, body := doJSON(t, router, http.MethodDelete, "/api/v1/sessions/"+sess.ID, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %v)", rec.Code, body)
	}

	got, err := s.Get(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.SessionID != "" {
		t.Fatalf("message.SessionID = %q, want empty after its session was deleted", got.SessionID)
	}
}

func TestAPI_ClearSessionsRequiresConfirm(t *testing.T) {
	s, _ := newTestStore(t)
	router := newRouter(t, s, Config{})

	rec, body := doJSON(t, router, http.MethodDelete, "/api/v1/sessions", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body: %v)", rec.Code, body)
	}
}

func TestAPI_ClearSessions(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	if err := s.CreateSession(ctx, sampleSession("10.0.0.1", "completed", time.Now())); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := s.CreateSession(ctx, sampleSession("10.0.0.2", "aborted", time.Now())); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	router, bus := newRouterWithBus(t, s, Config{})
	sub, unsub := bus.Subscribe()
	defer unsub()

	rec, body := doJSON(t, router, http.MethodDelete, "/api/v1/sessions?confirm=true", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %v)", rec.Code, body)
	}

	_, total, err := s.ListSessions(ctx, store.SessionListFilter{})
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if total != 0 {
		t.Fatalf("total = %d, want 0", total)
	}

	select {
	case ev := <-sub:
		if ev.Type != events.TypeSessionsCleared {
			t.Fatalf("got event type %q, want %q", ev.Type, events.TypeSessionsCleared)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for sessions.cleared event")
	}
}
