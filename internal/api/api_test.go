package api

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/0funct0ry/maelsink/internal/events"
	"github.com/0funct0ry/maelsink/internal/logging"
	"github.com/0funct0ry/maelsink/internal/store"
	"github.com/0funct0ry/maelsink/internal/store/sqlite"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func newTestStore(t *testing.T) (*sqlite.Store, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()

	db, err := sqlite.Open(ctx, filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	return sqlite.New(db, false, filepath.Join(dir, "attachments")), db
}

func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	logger, err := logging.New("error", "text", "")
	if err != nil {
		t.Fatalf("logging.New: %v", err)
	}
	return logger
}

func sampleMessage(subject, from, to string, receivedAt time.Time) *store.Message {
	raw := "From: " + from + "\r\nTo: " + to + "\r\nSubject: " + subject + "\r\n\r\nbody\r\n"
	return &store.Message{
		ReceivedAt: receivedAt,
		Size:       int64(len(raw)),
		Headers: []store.Header{
			{Name: "From", Value: from},
			{Name: "Subject", Value: subject},
		},
		From:      []store.Address{{Address: from}},
		To:        []store.Address{{Address: to}},
		Subject:   subject,
		TextBody:  "body",
		RawSource: []byte(raw),
		Attachments: []store.Attachment{
			{Filename: "doc.txt", ContentType: "text/plain", Size: 3, Data: []byte("abc")},
		},
	}
}

func newRouter(t *testing.T, s store.MessageStore, cfg Config) http.Handler {
	t.Helper()
	router, _ := newRouterWithBus(t, s, cfg)
	return router
}

func newRouterWithBus(t *testing.T, s store.MessageStore, cfg Config) (http.Handler, *events.Bus) {
	t.Helper()
	bus := events.NewBus()
	return New(s, bus, testLogger(t), cfg), bus
}

func doJSON(t *testing.T, router http.Handler, method, path string, headers map[string]string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var body map[string]any
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
	}
	return rec, body
}

func TestAPI_FullEndpointCoverage(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		msg := sampleMessage("subject", "alice@example.com", "bob@example.com", base.Add(time.Duration(i)*time.Hour))
		if err := s.Save(ctx, msg); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	router, bus := newRouterWithBus(t, s, Config{})

	t.Run("version", func(t *testing.T) {
		rec, body := doJSON(t, router, http.MethodGet, "/api/v1/version", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if _, ok := body["version"]; !ok {
			t.Fatalf("expected version field, got %+v", body)
		}
	})

	t.Run("health ok", func(t *testing.T) {
		rec, body := doJSON(t, router, http.MethodGet, "/api/v1/health", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if body["status"] != "ok" || body["db"] != "ok" {
			t.Fatalf("expected ok status, got %+v", body)
		}
	})

	t.Run("stats", func(t *testing.T) {
		rec, body := doJSON(t, router, http.MethodGet, "/api/v1/stats", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if int(body["total_messages"].(float64)) != 3 {
			t.Fatalf("expected 3 total messages, got %+v", body)
		}
	})

	t.Run("list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp listResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if resp.Total != 3 || len(resp.Messages) != 3 {
			t.Fatalf("expected 3 messages, got %+v", resp)
		}
		if resp.Messages[0].AttachmentCount != 1 || !resp.Messages[0].HasAttachments {
			t.Fatalf("expected attachment metadata populated, got %+v", resp.Messages[0])
		}
	})

	t.Run("list with malformed search query returns a friendly 400, not the raw SQLite error", func(t *testing.T) {
		rec, body := doJSON(t, router, http.MethodGet, "/api/v1/messages?q="+url.QueryEscape("certificate -urgent"), nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
		}
		errObj, ok := body["error"].(map[string]any)
		if !ok {
			t.Fatalf("expected an error envelope, got %+v", body)
		}
		if errObj["code"] != "invalid_query" {
			t.Fatalf("expected code invalid_query, got %+v", errObj)
		}
		msg, _ := errObj["message"].(string)
		for _, leaked := range []string{"sqlite", "SQL logic error", "fts5", "no such column"} {
			if strings.Contains(msg, leaked) {
				t.Fatalf("error message leaked internal detail %q: %s", leaked, msg)
			}
		}
	})

	msgs, _, err := s.List(ctx, store.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	id := msgs[0].ID

	t.Run("get", func(t *testing.T) {
		rec, body := doJSON(t, router, http.MethodGet, "/api/v1/messages/"+id, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if body["id"] != id {
			t.Fatalf("expected id %q, got %+v", id, body)
		}
	})

	t.Run("mark read", func(t *testing.T) {
		_, before := doJSON(t, router, http.MethodGet, "/api/v1/messages/"+id, nil)
		if before["read"] != false {
			t.Fatalf("expected new message unread, got %+v", before)
		}

		req := httptest.NewRequest(http.MethodPatch, "/api/v1/messages/"+id+"/read", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
		}

		_, body := doJSON(t, router, http.MethodGet, "/api/v1/messages/"+id, nil)
		if body["read"] != true {
			t.Fatalf("expected message marked read, got %+v", body)
		}
	})

	t.Run("mark unread", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/messages/"+id+"/read", strings.NewReader(`{"read":false}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
		}

		_, body := doJSON(t, router, http.MethodGet, "/api/v1/messages/"+id, nil)
		if body["read"] != false {
			t.Fatalf("expected message marked unread, got %+v", body)
		}

		// Restore to read for the remaining subtests in this run.
		req = httptest.NewRequest(http.MethodPatch, "/api/v1/messages/"+id+"/read", nil)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("mark read not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/messages/does-not-exist/read", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("get by id prefix", func(t *testing.T) {
		rec, body := doJSON(t, router, http.MethodGet, "/api/v1/messages/"+id[:8], nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if body["id"] != id {
			t.Fatalf("expected id %q, got %+v", id, body)
		}
	})

	t.Run("get not found", func(t *testing.T) {
		rec, body := doJSON(t, router, http.MethodGet, "/api/v1/messages/does-not-exist", nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
		errObj := body["error"].(map[string]any)
		if errObj["code"] != "message_not_found" {
			t.Fatalf("expected message_not_found, got %+v", body)
		}
	})

	t.Run("raw", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/"+id+"/raw", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if rec.Header().Get("Content-Type") != "message/rfc822" {
			t.Fatalf("expected message/rfc822, got %q", rec.Header().Get("Content-Type"))
		}
	})

	t.Run("export", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/"+id+"/export", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if rec.Header().Get("Content-Disposition") == "" {
			t.Fatalf("expected Content-Disposition header")
		}
	})

	t.Run("attachment download", func(t *testing.T) {
		full, err := s.Get(ctx, id)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		attID := full.Attachments[0].ID
		req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/"+id+"/attachments/"+attID, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if rec.Body.String() != "abc" {
			t.Fatalf("expected attachment bytes 'abc', got %q", rec.Body.String())
		}
	})

	t.Run("delete requires confirm for bulk", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/messages", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("delete single message", func(t *testing.T) {
		sub, unsub := bus.Subscribe()
		defer unsub()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/messages/"+id, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}
		if _, err := s.Get(ctx, id); err != store.ErrNotFound {
			t.Fatalf("expected message deleted")
		}

		select {
		case ev := <-sub:
			if ev.Type != events.TypeMessageDeleted {
				t.Fatalf("got event type %q, want %q", ev.Type, events.TypeMessageDeleted)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for message.deleted event")
		}
	})

	t.Run("bulk clear with confirm", func(t *testing.T) {
		sub, unsub := bus.Subscribe()
		defer unsub()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/messages?confirm=true", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}
		_, total, err := s.List(ctx, store.ListFilter{})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 0 {
			t.Fatalf("expected 0 messages after clear, got %d", total)
		}

		select {
		case ev := <-sub:
			if ev.Type != events.TypeMessagesCleared {
				t.Fatalf("got event type %q, want %q", ev.Type, events.TypeMessagesCleared)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for messages.cleared event")
		}
	})
}

func TestAPI_AmbiguousIDPrefix(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	msg1 := sampleMessage("one", "alice@example.com", "bob@example.com", time.Now())
	msg1.ID = "cccc1111cccc1111cccc1111"
	msg2 := sampleMessage("two", "alice@example.com", "bob@example.com", time.Now())
	msg2.ID = "cccc2222cccc2222cccc2222"
	if err := s.Save(ctx, msg1); err != nil {
		t.Fatalf("Save msg1: %v", err)
	}
	if err := s.Save(ctx, msg2); err != nil {
		t.Fatalf("Save msg2: %v", err)
	}

	router := newRouter(t, s, Config{})

	t.Run("get with ambiguous prefix", func(t *testing.T) {
		rec, body := doJSON(t, router, http.MethodGet, "/api/v1/messages/cccc", nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %+v", rec.Code, body)
		}
		errObj := body["error"].(map[string]any)
		if errObj["code"] != "ambiguous_id" {
			t.Fatalf("expected ambiguous_id, got %+v", body)
		}
	})

	t.Run("delete with ambiguous prefix leaves both messages", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/messages/cccc", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
		if _, err := s.Get(ctx, msg1.ID); err != nil {
			t.Fatalf("msg1 should survive a failed ambiguous delete: %v", err)
		}
		if _, err := s.Get(ctx, msg2.ID); err != nil {
			t.Fatalf("msg2 should survive a failed ambiguous delete: %v", err)
		}
	})
}

func TestAPI_PaginationFilterSort(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 7; i++ {
		from := "alice@example.com"
		if i == 6 {
			from = "carol@other.com"
		}
		msg := sampleMessage("subject", from, "bob@example.com", base.Add(time.Duration(i)*time.Hour))
		if err := s.Save(ctx, msg); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	router := newRouter(t, s, Config{})

	t.Run("paginates across two pages", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/messages?limit=3&offset=0", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var page1 listResponse
		json.Unmarshal(rec.Body.Bytes(), &page1)
		if len(page1.Messages) != 3 || page1.Total != 7 {
			t.Fatalf("expected page of 3 out of 7 total, got %+v", page1)
		}

		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/messages?limit=3&offset=3", nil)
		rec2 := httptest.NewRecorder()
		router.ServeHTTP(rec2, req2)
		var page2 listResponse
		json.Unmarshal(rec2.Body.Bytes(), &page2)
		if len(page2.Messages) != 3 {
			t.Fatalf("expected 3 more messages, got %+v", page2)
		}
		if page1.Messages[0].ID == page2.Messages[0].ID {
			t.Fatalf("expected different messages across pages")
		}
	})

	t.Run("filters by from", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/messages?from=carol", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var resp listResponse
		json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp.Total != 1 {
			t.Fatalf("expected 1 match, got %+v", resp)
		}
	})

	t.Run("sorts ascending", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/messages?sort=received_at_asc&limit=1", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var resp listResponse
		json.Unmarshal(rec.Body.Bytes(), &resp)
		if len(resp.Messages) != 1 || resp.Messages[0].ReceivedAt != base.UTC().Format(time.RFC3339) {
			t.Fatalf("expected oldest first, got %+v", resp.Messages)
		}
	})
}

func TestAPI_Auth(t *testing.T) {
	s, _ := newTestStore(t)

	t.Run("disabled allows unauthenticated", func(t *testing.T) {
		router := newRouter(t, s, Config{})
		rec, _ := doJSON(t, router, http.MethodGet, "/api/v1/version", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("enabled rejects missing key", func(t *testing.T) {
		router := newRouter(t, s, Config{Auth: Auth{Enabled: true, APIKey: "secret"}})
		rec, _ := doJSON(t, router, http.MethodGet, "/api/v1/version", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("enabled rejects wrong key", func(t *testing.T) {
		router := newRouter(t, s, Config{Auth: Auth{Enabled: true, APIKey: "secret"}})
		rec, _ := doJSON(t, router, http.MethodGet, "/api/v1/version", map[string]string{"Authorization": "Bearer wrong"})
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("enabled accepts correct key", func(t *testing.T) {
		router := newRouter(t, s, Config{Auth: Auth{Enabled: true, APIKey: "secret"}})
		rec, _ := doJSON(t, router, http.MethodGet, "/api/v1/version", map[string]string{"Authorization": "Bearer secret"})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

func TestAPI_BulkExportZip(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 2; i++ {
		msg := sampleMessage("subject", "alice@example.com", "bob@example.com", base.Add(time.Duration(i)*time.Hour))
		if err := s.Save(ctx, msg); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	router := newRouter(t, s, Config{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/export", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("expected application/zip, got %q", rec.Header().Get("Content-Type"))
	}

	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}
	if len(zr.File) != 2 {
		t.Fatalf("expected 2 entries in zip, got %d", len(zr.File))
	}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("opening zip entry %s: %v", f.Name, err)
		}
		raw, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("reading zip entry %s: %v", f.Name, err)
		}
		if _, err := mail.ReadMessage(bytes.NewReader(raw)); err != nil {
			t.Fatalf("entry %s is not a valid .eml: %v", f.Name, err)
		}
	}
}

func TestAPI_Health_DegradesOnClosedDB(t *testing.T) {
	s, db := newTestStore(t)
	router := newRouter(t, s, Config{})

	if err := db.Close(); err != nil {
		t.Fatalf("closing db: %v", err)
	}

	rec, body := doJSON(t, router, http.MethodGet, "/api/v1/health", nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if body["status"] == "ok" || body["db"] != "error" {
		t.Fatalf("expected degraded status, got %+v", body)
	}
}

func TestAPI_TagFilterAndTagsEndpoint(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	now := time.Now()
	m1 := sampleMessage("s1", "a@example.com", "b@example.com", now)
	m1.Tags = []string{"smoke", "release"}
	m2 := sampleMessage("s2", "a@example.com", "b@example.com", now)
	m2.Tags = []string{"smoke"}
	m3 := sampleMessage("s3", "a@example.com", "b@example.com", now)

	for _, m := range []*store.Message{m1, m2, m3} {
		if err := s.Save(ctx, m); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	router := newRouter(t, s, Config{})

	rec, body := doJSON(t, router, http.MethodGet, "/api/v1/messages?tag=smoke", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET ?tag=smoke = %d: %s", rec.Code, rec.Body.String())
	}
	if int(body["total"].(float64)) != 2 {
		t.Fatalf("total = %v, want 2", body["total"])
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/tags = %d: %s", rec.Code, rec.Body.String())
	}
	var tags []tagCountJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &tags); err != nil {
		t.Fatalf("unmarshal tags: %v", err)
	}
	counts := map[string]int{}
	for _, tc := range tags {
		counts[tc.Tag] = tc.Count
	}
	if counts["smoke"] != 2 || counts["release"] != 1 {
		t.Fatalf("tag counts = %+v, want smoke=2 release=1", counts)
	}

	rec, body = doJSON(t, router, http.MethodGet, "/api/v1/messages?tag=smoke&tag=release", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET ?tag=smoke&tag=release = %d: %s", rec.Code, rec.Body.String())
	}
	if int(body["total"].(float64)) != 2 {
		t.Fatalf("tags any total = %v, want 2", body["total"])
	}

	rec, body = doJSON(t, router, http.MethodGet, "/api/v1/messages?tag=smoke&tag=release&tag_mode=all", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET ?tag=smoke&tag=release&tag_mode=all = %d: %s", rec.Code, rec.Body.String())
	}
	if int(body["total"].(float64)) != 1 {
		t.Fatalf("tags all total = %v, want 1", body["total"])
	}

	rec, _ = doJSON(t, router, http.MethodGet, "/api/v1/messages?tag_mode=bogus", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("GET ?tag_mode=bogus = %d, want 400", rec.Code)
	}
}

func TestAPI_UpdateTags(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	now := time.Now()
	msg := sampleMessage("s1", "a@example.com", "b@example.com", now)
	msg.Tags = []string{"smoke"}
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	id := msg.ID

	router, bus := newRouterWithBus(t, s, Config{})

	t.Run("add and remove", func(t *testing.T) {
		sub, unsub := bus.Subscribe()
		defer unsub()

		req := httptest.NewRequest(http.MethodPatch, "/api/v1/messages/"+id+"/tags",
			strings.NewReader(`{"add":["release"],"remove":["smoke"]}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var body struct {
			Tags []string `json:"tags"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(body.Tags) != 1 || body.Tags[0] != "release" {
			t.Fatalf("response tags = %v, want [release]", body.Tags)
		}

		select {
		case ev := <-sub:
			if ev.Type != events.TypeMessageTagsUpdated {
				t.Fatalf("got event type %q, want %q", ev.Type, events.TypeMessageTagsUpdated)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for message.tags_updated event")
		}

		got, err := s.Get(ctx, id)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if len(got.Tags) != 1 || got.Tags[0] != "release" {
			t.Fatalf("stored tags = %v, want [release]", got.Tags)
		}
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/messages/does-not-exist/tags",
			strings.NewReader(`{"add":["x"]}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("empty tag is a validation error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/messages/"+id+"/tags",
			strings.NewReader(`{"add":["  "]}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/messages/"+id+"/tags",
			strings.NewReader(`not json`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestAPI_ReadHasAttachmentsParseWarningFilters(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	now := time.Now()

	unread := sampleMessage("u", "a@example.com", "b@example.com", now)
	unread.Attachments = nil
	if err := s.Save(ctx, unread); err != nil {
		t.Fatalf("Save: %v", err)
	}

	router := newRouter(t, s, Config{})

	rec, body := doJSON(t, router, http.MethodGet, "/api/v1/messages?read=false", nil)
	if rec.Code != http.StatusOK || int(body["total"].(float64)) != 1 {
		t.Fatalf("read=false: code=%d body=%+v", rec.Code, body)
	}

	rec, body = doJSON(t, router, http.MethodGet, "/api/v1/messages?has_attachments=true", nil)
	if rec.Code != http.StatusOK || int(body["total"].(float64)) != 0 {
		t.Fatalf("has_attachments=true: code=%d body=%+v", rec.Code, body)
	}

	rec, _ = doJSON(t, router, http.MethodGet, "/api/v1/messages?read=notabool", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("read=notabool: expected 400, got %d", rec.Code)
	}
}

func TestAPI_PreviewFieldOnSummary(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	msg := sampleMessage("s", "a@example.com", "b@example.com", time.Now())
	msg.TextBody = "hello world, this is the body of the message"
	if err := s.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	router := newRouter(t, s, Config{})
	rec, body := doJSON(t, router, http.MethodGet, "/api/v1/messages", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/messages = %d", rec.Code)
	}
	msgs := body["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	summary := msgs[0].(map[string]any)
	if summary["preview"] != "hello world, this is the body of the message" {
		t.Fatalf("preview = %v", summary["preview"])
	}
}
