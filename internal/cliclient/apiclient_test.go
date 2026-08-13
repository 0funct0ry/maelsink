package cliclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			if r.URL.Query().Get("confirm") != "true" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "confirmation_required", "message": "need confirm=true"}})
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		json.NewEncoder(w).Encode(ListResponse{
			Messages: []MessageSummary{{ID: "msg_1", From: "a@b.com", Subject: "hi"}},
			Total:    1, Limit: 50, Offset: 0,
		})
	})
	mux.HandleFunc("/api/v1/messages/msg_1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		json.NewEncoder(w).Encode(MessageDetail{MessageSummary: MessageSummary{ID: "msg_1", Subject: "hi"}})
	})
	mux.HandleFunc("/api/v1/messages/msg_1/export", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Subject: hi\r\n\r\nbody\r\n"))
	})
	mux.HandleFunc("/api/v1/messages/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "message_not_found", "message": "no message with id missing"}})
	})
	return httptest.NewServer(mux)
}

func TestClient_List(t *testing.T) {
	srv := newFixtureServer(t)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	resp, err := c.List(context.Background(), ListParams{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if resp.Total != 1 || len(resp.Messages) != 1 {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestClient_Get(t *testing.T) {
	srv := newFixtureServer(t)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	msg, err := c.Get(context.Background(), "msg_1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if msg.ID != "msg_1" {
		t.Errorf("ID = %q", msg.ID)
	}
}

func TestClient_Get_NotFound(t *testing.T) {
	srv := newFixtureServer(t)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	_, err := c.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("expected *HTTPError, got %T: %v", err, err)
	}
	if httpErr.Status != 404 || httpErr.Code != "message_not_found" {
		t.Errorf("httpErr = %+v", httpErr)
	}
}

func TestClient_Delete(t *testing.T) {
	srv := newFixtureServer(t)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	if err := c.Delete(context.Background(), "msg_1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestClient_Clear_RequiresConfirm(t *testing.T) {
	srv := newFixtureServer(t)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	if err := c.Clear(context.Background()); err != nil {
		t.Fatalf("Clear: %v", err)
	}
}

func TestClient_ExportRaw(t *testing.T) {
	srv := newFixtureServer(t)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	raw, err := c.ExportRaw(context.Background(), "msg_1")
	if err != nil {
		t.Fatalf("ExportRaw: %v", err)
	}
	if string(raw) != "Subject: hi\r\n\r\nbody\r\n" {
		t.Errorf("raw = %q", raw)
	}
}

func TestClient_ExportRawNamed_FallsBackToID(t *testing.T) {
	srv := newFixtureServer(t)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	raw, filename, err := c.ExportRawNamed(context.Background(), "msg_1")
	if err != nil {
		t.Fatalf("ExportRawNamed: %v", err)
	}
	if string(raw) != "Subject: hi\r\n\r\nbody\r\n" {
		t.Errorf("raw = %q", raw)
	}
	if filename != "msg_1" {
		t.Errorf("filename = %q, want fallback to id %q", filename, "msg_1")
	}
}

func TestClient_ExportRawNamed_UsesContentDisposition(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/messages/ab/export", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="ab3c7d06adb94e07f1a92725.eml"`)
		w.Write([]byte("Subject: hi\r\n\r\nbody\r\n"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	_, filename, err := c.ExportRawNamed(context.Background(), "ab")
	if err != nil {
		t.Fatalf("ExportRawNamed: %v", err)
	}
	if filename != "ab3c7d06adb94e07f1a92725.eml" {
		t.Errorf("filename = %q", filename)
	}
}

func TestClient_Unreachable(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "")
	_, err := c.List(context.Background(), ListParams{})
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*HTTPError); ok {
		t.Fatalf("expected a transport error, got *HTTPError: %v", err)
	}
}
