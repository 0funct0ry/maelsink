package api

import (
	"crypto/subtle"
	"log/slog"
	"strings"
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

// recoveryHandler logs a panic and returns the standard 500 error envelope
// instead of letting Gin's default recovery close the connection bare, so a
// single malformed request can never crash the process (SPEC.md §2.3).
func recoveryHandler(logger *slog.Logger) gin.RecoveryFunc {
	return func(c *gin.Context, recovered any) {
		logger.Error("http handler panic", "error", recovered, "path", c.Request.URL.Path)
		respondInternal(c, "internal server error")
	}
}

// authMiddleware enforces the optional bearer-token auth from SPEC.md §5.4.
// When cfg.Enabled is false, every request passes through unauthenticated.
func authMiddleware(cfg Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		const prefix = "Bearer "
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, prefix) {
			respondError(c, 401, "unauthorized", "missing or malformed Authorization header")
			return
		}
		token := strings.TrimPrefix(header, prefix)

		if subtle.ConstantTimeCompare([]byte(token), []byte(cfg.APIKey)) != 1 {
			respondError(c, 401, "unauthorized", "invalid API key")
			return
		}

		c.Next()
	}
}
