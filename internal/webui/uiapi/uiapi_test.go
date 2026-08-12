package uiapi

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/0funct0ry/maelsink/internal/api"
	"github.com/0funct0ry/maelsink/internal/config"
)

func newRouter(cfg Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterRoutes(&engine.RouterGroup, cfg)
	return engine
}

func TestInfo_ReturnsSMTPInfo(t *testing.T) {
	engine := newRouter(Config{SMTPHost: "127.0.0.1", SMTPPort: 2525})

	req := httptest.NewRequest("GET", "/ui-api/v1/info", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("GET /ui-api/v1/info = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != `{"smtp":{"host":"127.0.0.1","port":2525},"auth_enabled":false,"db_filename":""}` {
		t.Fatalf("unexpected body: %s", got)
	}
}

// TestInfo_ResolvesWildcardSMTPHost covers the abstraction-leakage-adjacent
// fix: a wildcard bind address (set via MAELSINK_SMTP_HOST=0.0.0.0 in
// Docker, per M9.0) must never be echoed back verbatim — it isn't
// something a client can dial. Resolve it to the request's own Host
// header instead.
func TestInfo_ResolvesWildcardSMTPHost(t *testing.T) {
	cases := []struct {
		name           string
		configuredHost string
		requestHost    string
		want           string
	}{
		{"wildcard resolves to request host", "0.0.0.0", "maelsink.example.com:8080", "maelsink.example.com"},
		{"wildcard IPv6 resolves to request host", "::", "maelsink.example.com:8080", "maelsink.example.com"},
		{"wildcard with no port on request host falls back to raw value", "0.0.0.0", "maelsink.example.com", "maelsink.example.com"},
		{"concrete host is echoed unchanged regardless of request host", "127.0.0.1", "unrelated.example.com:9090", "127.0.0.1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			engine := newRouter(Config{SMTPHost: tc.configuredHost, SMTPPort: 1025})

			req := httptest.NewRequest("GET", "/ui-api/v1/info", nil)
			req.Host = tc.requestHost
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)

			if rec.Code != 200 {
				t.Fatalf("GET /ui-api/v1/info = %d, want 200: %s", rec.Code, rec.Body.String())
			}
			wantSubstr := `"host":"` + tc.want + `"`
			if got := rec.Body.String(); !strings.Contains(got, wantSubstr) {
				t.Fatalf("body = %s, want it to contain %s", got, wantSubstr)
			}
		})
	}
}

// TestInfo_ReturnsDBFilename covers the display-accuracy fix for the
// Sidebar's storage widget: db_filename should mirror storage.path's
// already-basename-reduced value from ConfigEntries (M8.7's redaction),
// not a hardcoded literal.
func TestInfo_ReturnsDBFilename(t *testing.T) {
	t.Run("present in ConfigEntries", func(t *testing.T) {
		entries := []config.Entry{
			{Section: "storage", Key: "storage.path", Value: "custom-name.db"},
		}
		engine := newRouter(Config{ConfigEntries: entries})

		req := httptest.NewRequest("GET", "/ui-api/v1/info", nil)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		if got := rec.Body.String(); !strings.Contains(got, `"db_filename":"custom-name.db"`) {
			t.Fatalf("body = %s, want db_filename custom-name.db", got)
		}
	})

	t.Run("absent from ConfigEntries", func(t *testing.T) {
		engine := newRouter(Config{ConfigEntries: nil})

		req := httptest.NewRequest("GET", "/ui-api/v1/info", nil)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)

		if got := rec.Body.String(); !strings.Contains(got, `"db_filename":""`) {
			t.Fatalf("body = %s, want empty db_filename", got)
		}
	})
}

func TestInfo_RequiresAuthWhenEnabled(t *testing.T) {
	engine := newRouter(Config{Auth: api.Auth{Enabled: true, APIKey: "secret"}})

	req := httptest.NewRequest("GET", "/ui-api/v1/info", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}

	req = httptest.NewRequest("GET", "/ui-api/v1/info", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200 with correct token, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestConfig_ReturnsEntriesAndExcludesSecrets(t *testing.T) {
	cfg := config.Defaults()
	cfg.SMTP.Auth.Password = "supersecret"
	cfg.API.Auth.APIKey = "topsecretkey"
	entries := config.Dump(cfg, config.Provenance{})

	engine := newRouter(Config{ConfigEntries: entries})

	req := httptest.NewRequest("GET", "/ui-api/v1/config", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("GET /ui-api/v1/config = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "supersecret") || strings.Contains(body, "topsecretkey") {
		t.Fatalf("response leaked a secret value: %s", body)
	}
	if strings.Contains(body, "smtp.auth.password") || strings.Contains(body, "api.auth.api_key") {
		t.Fatalf("response included a secret key: %s", body)
	}
	if !strings.Contains(body, `"smtp.host"`) {
		t.Fatalf("response missing expected key smtp.host: %s", body)
	}
}

func TestInfo_BasePath(t *testing.T) {
	engine := newRouter(Config{BasePath: "/maelsink"})

	req := httptest.NewRequest("GET", "/maelsink/ui-api/v1/info", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("GET /maelsink/ui-api/v1/info = %d, want 200", rec.Code)
	}
}
