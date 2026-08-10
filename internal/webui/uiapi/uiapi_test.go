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
	engine := newRouter(Config{SMTPHost: "0.0.0.0", SMTPPort: 2525})

	req := httptest.NewRequest("GET", "/ui-api/v1/info", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("GET /ui-api/v1/info = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != `{"smtp":{"host":"0.0.0.0","port":2525},"auth_enabled":false}` {
		t.Fatalf("unexpected body: %s", got)
	}
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
