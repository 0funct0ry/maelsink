// Package webui embeds and serves the maelsink Web UI SPA (SPEC.md §8),
// mounting the /api/v1 surface read-through (SPEC.md §5) via
// internal/api.RegisterRoutes, and the WebSocket hub (SPEC.md §5.5, M7.0)
// via internal/ws. It depends only on the store.MessageStore interface,
// internal/api's route registration, and internal/events/internal/ws —
// never the other way around (SPEC.md §2.3 point 4).
package webui

import (
	"bytes"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/0funct0ry/maelsink/internal/api"
	"github.com/0funct0ry/maelsink/internal/config"
	"github.com/0funct0ry/maelsink/internal/events"
	"github.com/0funct0ry/maelsink/internal/store"
	"github.com/0funct0ry/maelsink/internal/webui/uiapi"
	"github.com/0funct0ry/maelsink/internal/ws"
)

//go:embed all:dist
var distFS embed.FS

const (
	basePathPlaceholder = "__MAELSINK_BASE_PATH__"
	baseHrefPlaceholder = "__MAELSINK_BASE_HREF__"
)

// Config configures the router returned by New.
type Config struct {
	// BasePath is the configured reverse-proxy subpath (e.g. "/maelsink"),
	// per SPEC.md §3.4. Empty means "serve at root", falling back per
	// request to the X-Forwarded-Prefix header when unset.
	BasePath string
	Auth     api.Auth

	// WebAuthFile, if non-empty, is the path to an htpasswd-style basic-auth
	// file gating every route on this router except /api/v1 (SPEC.md §5.4 vs
	// the Web UI's own Basic Auth wall, M8.8). Empty disables the gate
	// entirely — no middleware is installed, zero overhead.
	WebAuthFile string

	// SMTPHost/SMTPPort are surfaced read-only via /ui-api/v1/info for the
	// Inbox empty state and Settings screen (SPEC.md §8.1) — the Web UI has
	// no other way to learn the SMTP listener's address.
	SMTPHost string
	SMTPPort int

	// ConfigEntries backs GET /ui-api/v1/config (M6.1): every non-secret
	// config key's resolved value plus its provenance, precomputed once at
	// startup (config values are immutable for the process lifetime).
	ConfigEntries []config.Entry
}

// New builds the Web UI router: the embedded SPA under cfg.BasePath, with
// the /api/v1 surface mounted read-through at the same base path, plus the
// GET /ws WebSocket endpoint (SPEC.md §5.5) served by hub.
func New(messageStore store.MessageStore, bus *events.Bus, hub *ws.Hub, logger *slog.Logger, cfg Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()
	engine.Use(requestLoggingMiddleware(logger), gin.CustomRecovery(recoveryHandler(logger)))
	if cfg.WebAuthFile != "" {
		engine.Use(basicAuthMiddleware(cfg.WebAuthFile, cfg.BasePath))
	}

	api.RegisterRoutes(&engine.RouterGroup, messageStore, bus, logger, api.Config{
		BasePath: cfg.BasePath,
		Auth:     cfg.Auth,
	})
	uiapi.RegisterRoutes(&engine.RouterGroup, uiapi.Config{
		BasePath:      cfg.BasePath,
		Auth:          cfg.Auth,
		SMTPHost:      cfg.SMTPHost,
		SMTPPort:      cfg.SMTPPort,
		ConfigEntries: cfg.ConfigEntries,
	})
	engine.GET(cfg.BasePath+"/ws", hub.ServeWS)

	assets, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("webui: embedded dist directory missing: " + err.Error())
	}
	indexHTML, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		panic("webui: embedded index.html missing: " + err.Error())
	}

	registerSPARoutes(engine, assets, indexHTML, cfg.BasePath, hub)

	return engine
}

// registerSPARoutes serves static assets byte-for-byte from assets, and
// falls back to a base-path-templated index.html for every other GET so the
// SPA's client-side router can take over (SPEC.md §3.4).
//
// When configuredBasePath is empty, /ws is already mounted at the engine
// level by New, but that only covers the root-mounted case: an operator
// relying on the zero-config X-Forwarded-Prefix fallback (SPEC.md §3.4)
// expects the WS endpoint to live under that forwarded prefix instead, e.g.
// /maelsink/ws. Since the prefix isn't known until request time, that case
// is handled here, in NoRoute, using the same resolveBasePath-stripped
// reqPath already computed for asset/index resolution.
func registerSPARoutes(engine *gin.Engine, assets fs.FS, indexHTML []byte, configuredBasePath string, hub *ws.Hub) {
	fileServer := http.FileServer(http.FS(assets))

	serveIndex := func(c *gin.Context) {
		base := resolveBasePath(configuredBasePath, c.Request.Header.Get("X-Forwarded-Prefix"))
		c.Data(http.StatusOK, "text/html; charset=utf-8", renderIndex(indexHTML, base))
	}

	engine.GET(configuredBasePath+"/", serveIndex)
	engine.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet {
			c.Status(http.StatusNotFound)
			return
		}

		base := resolveBasePath(configuredBasePath, c.Request.Header.Get("X-Forwarded-Prefix"))
		reqPath := strings.TrimPrefix(c.Request.URL.Path, base)
		reqPath = strings.TrimPrefix(reqPath, "/")

		if configuredBasePath == "" && reqPath == "ws" {
			hub.ServeWS(c)
			return
		}

		if reqPath != "" {
			if f, err := assets.Open(reqPath); err == nil {
				_ = f.Close()
				r2 := c.Request.Clone(c.Request.Context())
				r2.URL.Path = "/" + reqPath
				fileServer.ServeHTTP(c.Writer, r2)
				return
			}
		}

		// Not a real asset: an SPA client-side route (e.g. /messages/abc) or
		// the bare root — serve the templated index.html so the frontend
		// router takes over instead of a 404.
		serveIndex(c)
	})
}

// resolveBasePath applies SPEC.md §3.4's precedence: explicit configured
// base path wins, then X-Forwarded-Prefix, then root.
func resolveBasePath(configured, forwardedPrefix string) string {
	if configured != "" {
		return configured
	}
	if forwardedPrefix != "" {
		return forwardedPrefix
	}
	return ""
}

// renderIndex rewrites the embedded index.html's base-path placeholders to
// base, per request. Only these two placeholders are ever templated —
// everything else in index.html is served byte-for-byte from the build.
func renderIndex(indexHTML []byte, base string) []byte {
	href := base + "/"
	out := bytes.ReplaceAll(indexHTML, []byte(baseHrefPlaceholder), []byte(href))
	out = bytes.ReplaceAll(out, []byte(basePathPlaceholder), []byte(base))
	return out
}
