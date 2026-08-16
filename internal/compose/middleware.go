package compose

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// requestLoggingMiddleware logs method, path, status, and latency for every
// request through logger. A small local copy of internal/webui's equivalent
// helper — duplicated, not imported, since internal/compose must not depend
// on internal/webui (SPEC.md §2.3).
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
// Gin's default recovery close the connection.
func recoveryHandler(logger *slog.Logger) gin.RecoveryFunc {
	return func(c *gin.Context, recovered any) {
		logger.Error("http handler panic", "error", recovered, "path", c.Request.URL.Path)
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}
