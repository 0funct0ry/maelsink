// Package compose implements the `maelsink compose` local server (SPEC.md
// §7.7): a standalone HTTP server serving the compose SPA and proxying a
// curated slice of a target maelsink instance's REST API under
// /compose-api/v1/*, injecting target credentials server-side so the
// browser never sees them.
//
// Like internal/shell, this is a leaf package (SPEC.md §2.3): it depends
// only on internal/cliclient, internal/logging, internal/version, and
// nothing depends on it. It must never import internal/store,
// internal/smtp, internal/api, or internal/webui.
package compose

import (
	"io/fs"
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/compose/job"
)

// Config configures the router returned by New. Left intentionally minimal
// for M13.0 — compose has no BasePath/reverse-proxy story or auth wall of
// its own in this milestone.
type Config struct{}

// New builds compose's router: the /compose-api/v1/* target-proxy handlers
// plus the embedded SPA, served at the root (no BasePath support yet).
// target carries the SMTP credentials the Composer's stateless /send
// handler (and the Jobs Panel's job runners, M13.3) dial with (SPEC.md
// §7.7.4.1) — separate from client, which only talks to the target's REST
// API. mgr is compose's job manager (SPEC.md §7.7.4.3): the one piece of
// real server-side state compose owns, lost on process restart by design.
func New(client *cliclient.Client, logger *slog.Logger, target TargetConfig, mgr *job.Manager, cfg Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(requestLoggingMiddleware(logger), gin.CustomRecovery(recoveryHandler(logger)))

	rg := engine.Group("/compose-api/v1")
	rg.GET("/health", healthHandler(client))
	rg.GET("/messages", listMessagesHandler(client))
	rg.GET("/messages/:id", getMessageHandler(client))
	rg.DELETE("/messages/:id", deleteMessageHandler(client))
	rg.DELETE("/messages", clearMessagesHandler(client))
	rg.POST("/render", renderHandler())
	rg.POST("/send", sendHandler(target))
	rg.GET("/functions", functionsHandler())
	rg.GET("/stats", statsHandler(client))
	rg.GET("/version", versionHandler(client))
	rg.GET("/export", exportHandler(client))
	rg.GET("/messages/:id/attachments/:attachmentId", attachmentHandler(client))
	rg.GET("/jobs", listJobsHandler(mgr))
	rg.POST("/jobs/:kind", startJobHandler(mgr, target))
	rg.GET("/jobs/:kind/stream", jobStreamHandler(mgr))
	rg.POST("/jobs/:kind/cancel", cancelJobHandler(mgr))

	assets, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("compose: embedded dist directory missing: " + err.Error())
	}
	indexHTML, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		panic("compose: embedded index.html missing: " + err.Error())
	}
	registerSPARoutes(engine, assets, indexHTML)

	return engine
}
