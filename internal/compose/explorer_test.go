package compose

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0funct0ry/maelsink/internal/cliclient"
)

func TestStatsHandler(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cliclient.Stats{TotalMessages: 3, TotalSizeBytes: 1024})
	}))
	defer target.Close()

	client := newTestClient(t, target)
	engine := New(client, testLogger(), TargetConfig{}, Config{})

	rec := doRequest(t, engine, http.MethodGet, "/compose-api/v1/stats")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	var stats cliclient.Stats
	if err := json.Unmarshal(rec.Body.Bytes(), &stats); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if stats.TotalMessages != 3 {
		t.Fatalf("TotalMessages = %d, want 3", stats.TotalMessages)
	}

	target.Close()
	rec = doRequest(t, engine, http.MethodGet, "/compose-api/v1/stats")
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 on transport error", rec.Code)
	}
}

func TestVersionHandler(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cliclient.VersionInfo{Version: "1.2.3"})
	}))
	defer target.Close()

	client := newTestClient(t, target)
	engine := New(client, testLogger(), TargetConfig{}, Config{})

	rec := doRequest(t, engine, http.MethodGet, "/compose-api/v1/version")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	var body versionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Target == nil || body.Target.Version != "1.2.3" {
		t.Fatalf("body.Target = %+v, want Version 1.2.3", body.Target)
	}
	if body.Compose.Go == "" {
		t.Fatalf("body.Compose = %+v, want non-empty Go field", body.Compose)
	}
}

func TestExportHandler(t *testing.T) {
	const zipBody = "PK\x03\x04fake-zip-bytes"

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/messages/export" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("subject") != "hi" {
			t.Errorf("subject query = %q, want hi", r.URL.Query().Get("subject"))
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write([]byte(zipBody))
	}))
	defer target.Close()

	client := newTestClient(t, target)
	engine := New(client, testLogger(), TargetConfig{}, Config{})

	rec := doRequest(t, engine, http.MethodGet, "/compose-api/v1/export?subject=hi")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename="export.zip"` {
		t.Fatalf("Content-Disposition = %q", got)
	}
	if rec.Body.String() != zipBody {
		t.Fatalf("body = %q, want %q", rec.Body.String(), zipBody)
	}

	target.Close()
	rec = doRequest(t, engine, http.MethodGet, "/compose-api/v1/export")
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 on transport error", rec.Code)
	}
}

func TestAttachmentHandler(t *testing.T) {
	const attBody = "raw attachment bytes"

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/messages/notfound/attachments/att1" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"no such attachment"}}`))
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", `attachment; filename="notes.txt"`)
		_, _ = w.Write([]byte(attBody))
	}))
	defer target.Close()

	client := newTestClient(t, target)
	engine := New(client, testLogger(), TargetConfig{}, Config{})

	rec := doRequest(t, engine, http.MethodGet, "/compose-api/v1/messages/abc/attachments/att1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename="notes.txt"` {
		t.Fatalf("Content-Disposition = %q", got)
	}
	if rec.Body.String() != attBody {
		t.Fatalf("body = %q, want %q", rec.Body.String(), attBody)
	}

	rec = doRequest(t, engine, http.MethodGet, "/compose-api/v1/messages/notfound/attachments/att1")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
