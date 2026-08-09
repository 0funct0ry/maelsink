package webui

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// requestLoggingMiddleware logs method, path, status, and latency for every
// request through logger, per SPEC.md §10.
func requestLoggingMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		logger.Info("http request",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
		)
	}
}

// recoveryHandler logs a panic and returns a bare 500 instead of letting
// Gin's default recovery close the connection, so a single malformed
// request can never crash the process (SPEC.md §2.3).
func recoveryHandler(logger *slog.Logger) gin.RecoveryFunc {
	return func(c *gin.Context, recovered any) {
		logger.Error("http handler panic", "error", recovered, "path", c.Request.URL.Path)
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}
