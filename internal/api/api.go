// Package api implements maelsink's stable REST surface (SPEC.md §5) under
// /api/v1. Per SPEC.md §2.3 point 3, this package depends only on the
// store.MessageStore interface — never on a concrete storage backend or on
// any Web UI / internal-API package — so it can be mounted standalone on the
// dedicated REST API port, and read-through on the Web UI port, without
// pulling in UI-only concerns.
package api

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/0funct0ry/maelsink/internal/store"
)

// Auth configures the optional bearer-token auth middleware (SPEC.md §5.4).
type Auth struct {
	Enabled bool
	APIKey  string
}

// Config configures the router returned by New.
type Config struct {
	// BasePath is prefixed to every /api/v1 route, per SPEC.md §3.4's
	// reverse-proxy subpath support (e.g. "/maelsink").
	BasePath string
	Auth     Auth
}

// New builds the /api/v1 router against store, logging requests through
// logger (SPEC.md §10) and enforcing cfg.Auth when enabled.
func New(messageStore store.MessageStore, logger *slog.Logger, cfg Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()
	engine.Use(requestLoggingMiddleware(logger), gin.CustomRecovery(recoveryHandler(logger)))

	RegisterRoutes(&engine.RouterGroup, messageStore, cfg)

	return engine
}

// RegisterRoutes mounts the /api/v1 surface onto rg (typically an
// engine.Group(basePath)), enforcing cfg.Auth. Exported so internal/webui
// (M5.0) can mount the same routes read-through on the Web UI port
// (SPEC.md §5) without duplicating the route table — this is the single
// source of truth for /api/v1's shape.
func RegisterRoutes(rg *gin.RouterGroup, messageStore store.MessageStore, cfg Config) {
	h := &handlers{store: messageStore}

	v1 := rg.Group(cfg.BasePath + "/api/v1")
	v1.Use(authMiddleware(cfg.Auth))
	{
		v1.GET("/messages", h.listMessages)
		v1.GET("/messages/export", h.bulkExport)
		v1.DELETE("/messages", h.clearMessages)
		v1.GET("/messages/:id", h.getMessage)
		v1.PATCH("/messages/:id/read", h.markRead)
		v1.DELETE("/messages/:id", h.deleteMessage)
		v1.GET("/messages/:id/raw", h.rawMessage)
		v1.GET("/messages/:id/export", h.exportMessage)
		v1.GET("/messages/:id/attachments/:attId", h.getAttachment)
		v1.GET("/stats", h.stats)
		v1.GET("/tags", h.listTags)
		v1.GET("/health", h.health)
		v1.GET("/version", h.version)
	}
}
