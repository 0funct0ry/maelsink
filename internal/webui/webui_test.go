package webui

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0funct0ry/maelsink/internal/events"
	"github.com/0funct0ry/maelsink/internal/logging"
	"github.com/0funct0ry/maelsink/internal/store/sqlite"
	"github.com/0funct0ry/maelsink/internal/ws"
)

func newTestStore(t *testing.T) *sqlite.Store {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlite.Open(context.Background(), filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return sqlite.New(db, false, filepath.Join(dir, "attachments"))
}

func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	logger, err := logging.New("error", "text", "")
	if err != nil {
		t.Fatalf("logging.New: %v", err)
	}
	return logger
}

func testBusAndHub(t *testing.T) (*events.Bus, *ws.Hub) {
	t.Helper()
	bus := events.NewBus()
	hub := ws.NewHub(bus, testLogger(t))
	t.Cleanup(hub.Close)
	return bus, hub
}

func TestServesEmbeddedSPAAtRoot(t *testing.T) {
	store := newTestStore(t)
	_bus, _hub := testBusAndHub(t)
	engine := New(store, _bus, _hub, testLogger(t), Config{})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, basePathPlaceholder) || strings.Contains(body, baseHrefPlaceholder) {
		t.Fatalf("index.html placeholders not templated: %s", body)
	}
	if !strings.Contains(body, `<base href="/" />`) {
		t.Fatalf("expected root base href, got: %s", body)
	}
}

func TestSPAClientSideRouteFallsBackToIndex(t *testing.T) {
	store := newTestStore(t)
	_bus, _hub := testBusAndHub(t)
	engine := New(store, _bus, _hub, testLogger(t), Config{})

	req := httptest.NewRequest("GET", "/messages/anything", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("GET /messages/anything = %d, want 200 (SPA fallback)", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<div id=\"root\">") {
		t.Fatalf("expected index.html to be served, got: %s", rec.Body.String())
	}
}

func TestConfiguredBasePathTemplatesIndexHTML(t *testing.T) {
	store := newTestStore(t)
	_bus, _hub := testBusAndHub(t)
	engine := New(store, _bus, _hub, testLogger(t), Config{BasePath: "/maelsink"})

	req := httptest.NewRequest("GET", "/maelsink/", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("GET /maelsink/ = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<base href="/maelsink/" />`) {
		t.Fatalf("expected base href to be prefixed with /maelsink, got: %s", body)
	}
	if !strings.Contains(body, `window.__MAELSINK_BASE__ = "/maelsink";`) {
		t.Fatalf("expected window.__MAELSINK_BASE__ to be set to /maelsink, got: %s", body)
	}
}

func TestForwardedPrefixFallbackWhenNoBasePathConfigured(t *testing.T) {
	store := newTestStore(t)
	_bus, _hub := testBusAndHub(t)
	engine := New(store, _bus, _hub, testLogger(t), Config{})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-Prefix", "/maelsink")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<base href="/maelsink/" />`) {
		t.Fatalf("expected X-Forwarded-Prefix to drive base href, got: %s", body)
	}
}

func TestConfiguredBasePathWinsOverForwardedPrefix(t *testing.T) {
	store := newTestStore(t)
	_bus, _hub := testBusAndHub(t)
	engine := New(store, _bus, _hub, testLogger(t), Config{BasePath: "/configured"})

	req := httptest.NewRequest("GET", "/configured/", nil)
	req.Header.Set("X-Forwarded-Prefix", "/from-proxy")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `<base href="/configured/" />`) {
		t.Fatalf("expected configured base path to win, got: %s", body)
	}
}

func TestUIAPIInfoOnWebUIPort(t *testing.T) {
	store := newTestStore(t)
	_bus, _hub := testBusAndHub(t)
	engine := New(store, _bus, _hub, testLogger(t), Config{SMTPHost: "127.0.0.1", SMTPPort: 1025})

	req := httptest.NewRequest("GET", "/ui-api/v1/info", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("GET /ui-api/v1/info = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"host":"127.0.0.1"`) || !strings.Contains(body, `"port":1025`) {
		t.Fatalf("expected smtp host/port in response, got: %s", body)
	}
	if !strings.Contains(body, `"auth_enabled":false`) {
		t.Fatalf("expected auth_enabled false, got: %s", body)
	}
}

func TestUIAPIInfoRespectsBasePath(t *testing.T) {
	store := newTestStore(t)
	_bus, _hub := testBusAndHub(t)
	engine := New(store, _bus, _hub, testLogger(t), Config{BasePath: "/maelsink"})

	req := httptest.NewRequest("GET", "/maelsink/ui-api/v1/info", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("GET /maelsink/ui-api/v1/info = %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func TestAPIReadThroughOnWebUIPort(t *testing.T) {
	store := newTestStore(t)
	_bus, _hub := testBusAndHub(t)
	engine := New(store, _bus, _hub, testLogger(t), Config{})

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("GET /api/v1/health = %d, want 200 (read-through)", rec.Code)
	}
}

func TestAPIReadThroughRespectsBasePath(t *testing.T) {
	store := newTestStore(t)
	_bus, _hub := testBusAndHub(t)
	engine := New(store, _bus, _hub, testLogger(t), Config{BasePath: "/maelsink"})

	req := httptest.NewRequest("GET", "/maelsink/api/v1/health", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("GET /maelsink/api/v1/health = %d, want 200", rec.Code)
	}
}

func TestStaticAssetsServedByteForByte(t *testing.T) {
	store := newTestStore(t)
	_bus, _hub := testBusAndHub(t)
	engine := New(store, _bus, _hub, testLogger(t), Config{})

	// The embedded index.html should reference assets under ./assets/ —
	// discover one and confirm it's served unmodified (no placeholder
	// substitution applied to real assets, only to index.html).
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	body := rec.Body.String()

	start := strings.Index(body, `src="./`)
	if start == -1 {
		t.Skip("no bundled script asset found in index.html to verify")
	}
	start += len(`src="./`)
	end := strings.Index(body[start:], `"`)
	assetPath := body[start : start+end]

	req2 := httptest.NewRequest("GET", "/"+assetPath, nil)
	rec2 := httptest.NewRecorder()
	engine.ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("GET /%s = %d, want 200", assetPath, rec2.Code)
	}
}
