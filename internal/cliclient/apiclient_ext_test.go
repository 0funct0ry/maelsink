package cliclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newExtFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/stats", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"total_messages":     3,
			"total_size_bytes":   1024,
			"oldest_received_at": "2026-01-01T00:00:00Z",
			"newest_received_at": "2026-01-02T00:00:00Z",
		})
	})
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("degraded") == "true" {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "degraded", "db": "error", "smtp": "listening"})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "db": "ok", "smtp": "listening"})
	})
	mux.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"version": "1.2.3", "commit": "abc123", "go": "go1.26.4"})
	})
	mux.HandleFunc("/api/v1/messages/msg_1/attachments/att_1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Disposition", `attachment; filename="logo.png"`)
		w.Write([]byte("fake-png-bytes"))
	})
	mux.HandleFunc("/api/v1/messages/export", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="messages.zip"`)
		w.Write([]byte("PK\x03\x04fake-zip-bytes"))
	})
	return httptest.NewServer(mux)
}

func TestClient_Stats(t *testing.T) {
	srv := newExtFixtureServer(t)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	stats, err := c.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.TotalMessages != 3 || stats.TotalSizeBytes != 1024 {
		t.Errorf("stats = %+v", stats)
	}
	if stats.OldestReceivedAt == nil || *stats.OldestReceivedAt != "2026-01-01T00:00:00Z" {
		t.Errorf("OldestReceivedAt = %v", stats.OldestReceivedAt)
	}
}

func TestClient_Health_OK(t *testing.T) {
	srv := newExtFixtureServer(t)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	h, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if h.Status != "ok" {
		t.Errorf("Status = %q", h.Status)
	}
}

func TestClient_Health_Degraded(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "degraded", "db": "error", "smtp": "listening"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	h, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("Health (degraded): %v", err)
	}
	if h.Status != "degraded" || h.DB != "error" {
		t.Errorf("h = %+v", h)
	}
}

func TestClient_Version(t *testing.T) {
	srv := newExtFixtureServer(t)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	v, err := c.Version(context.Background())
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if v.Version != "1.2.3" || v.Commit != "abc123" {
		t.Errorf("v = %+v", v)
	}
}

func TestClient_Attachment(t *testing.T) {
	srv := newExtFixtureServer(t)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	data, contentType, filename, err := c.Attachment(context.Background(), "msg_1", "att_1")
	if err != nil {
		t.Fatalf("Attachment: %v", err)
	}
	if string(data) != "fake-png-bytes" {
		t.Errorf("data = %q", data)
	}
	if contentType != "image/png" {
		t.Errorf("contentType = %q", contentType)
	}
	if filename != "logo.png" {
		t.Errorf("filename = %q", filename)
	}
}

func TestClient_BulkExport(t *testing.T) {
	srv := newExtFixtureServer(t)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	data, err := c.BulkExport(context.Background(), ListParams{})
	if err != nil {
		t.Fatalf("BulkExport: %v", err)
	}
	if string(data) != "PK\x03\x04fake-zip-bytes" {
		t.Errorf("data = %q", data)
	}
}

func TestClient_Stats_Unreachable(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "")
	_, err := c.Stats(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*HTTPError); ok {
		t.Fatalf("expected a transport error, got *HTTPError: %v", err)
	}
}
