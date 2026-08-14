package webui

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/0funct0ry/maelsink/internal/webauth"
)

func TestBasicAuth_NoFileConfigured_AllowsUnauthenticated(t *testing.T) {
	store := newTestStore(t)
	bus, hub := testBusAndHub(t)
	engine := New(store, bus, hub, testLogger(t), Config{})

	req := httptest.NewRequest("GET", "/ui-api/v1/info", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("GET /ui-api/v1/info with no auth file = %d, want 200", rec.Code)
	}
}

func TestBasicAuth_FileConfigured_RequiresCredentials(t *testing.T) {
	authFile := filepath.Join(t.TempDir(), "webauth.htpasswd")
	if err := webauth.Upsert(authFile, "alice", "s3cret"); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	store := newTestStore(t)
	bus, hub := testBusAndHub(t)
	engine := New(store, bus, hub, testLogger(t), Config{WebAuthFile: authFile})

	t.Run("no credentials -> 401 with WWW-Authenticate", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ui-api/v1/info", nil)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		if rec.Code != 401 {
			t.Fatalf("GET /ui-api/v1/info with no creds = %d, want 401", rec.Code)
		}
		if got := rec.Header().Get("WWW-Authenticate"); got != `Basic realm="maelsink"` {
			t.Fatalf("WWW-Authenticate = %q", got)
		}
	})

	t.Run("correct credentials -> 200", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ui-api/v1/info", nil)
		req.SetBasicAuth("alice", "s3cret")
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		if rec.Code != 200 {
			t.Fatalf("GET /ui-api/v1/info with correct creds = %d, want 200", rec.Code)
		}
	})

	t.Run("incorrect credentials -> 401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ui-api/v1/info", nil)
		req.SetBasicAuth("alice", "wrong")
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		if rec.Code != 401 {
			t.Fatalf("GET /ui-api/v1/info with wrong creds = %d, want 401", rec.Code)
		}
	})

	t.Run("/api/v1 reachable without basic auth credentials", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/version", nil)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		if rec.Code == 401 {
			t.Fatalf("GET /api/v1/version without basic auth = %d, want not 401", rec.Code)
		}
	})

	t.Run("static assets (SPA shell) require basic auth credentials", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		if rec.Code != 401 {
			t.Fatalf("GET / without basic auth = %d, want 401 (static assets are gated)", rec.Code)
		}
	})
}
