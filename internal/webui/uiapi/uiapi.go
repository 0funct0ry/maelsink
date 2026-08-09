// Package uiapi holds the Web UI's internal-only /ui-api/v1/* endpoints
// (SPEC.md §2.3 point 4) — data the Web UI's screens need that has no
// business living in the stable /api/v1 surface (internal/api), such as the
// SMTP connection info shown on the Inbox empty state and the Settings
// screen (SPEC.md §8.1).
package uiapi

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/0funct0ry/maelsink/internal/api"
)

// Config configures the router built by RegisterRoutes.
type Config struct {
	// BasePath is prefixed to every /ui-api/v1 route, mirroring
	// internal/api.Config's reverse-proxy subpath support (SPEC.md §3.4).
	BasePath string
	Auth     api.Auth

	SMTPHost string
	SMTPPort int
}

type smtpInfo struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type infoResponse struct {
	SMTP        smtpInfo `json:"smtp"`
	AuthEnabled bool     `json:"auth_enabled"`
}

// RegisterRoutes mounts /ui-api/v1 onto rg, gated by the same bearer-token
// auth as /api/v1 (cfg.Auth) so the two surfaces have one uniform auth story.
func RegisterRoutes(rg *gin.RouterGroup, cfg Config) {
	v1 := rg.Group(cfg.BasePath + "/ui-api/v1")
	v1.Use(authMiddleware(cfg.Auth))
	{
		v1.GET("/info", func(c *gin.Context) {
			c.JSON(http.StatusOK, infoResponse{
				SMTP:        smtpInfo{Host: cfg.SMTPHost, Port: cfg.SMTPPort},
				AuthEnabled: cfg.Auth.Enabled,
			})
		})
	}
}

// authMiddleware mirrors internal/api's unexported authMiddleware: when
// disabled it's a no-op, otherwise it requires a matching "Bearer <key>"
// Authorization header. Duplicated rather than imported since internal/api
// doesn't export its middleware, and this package must not depend on
// internal/api beyond the shared Auth config type.
func authMiddleware(cfg api.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		const prefix = "Bearer "
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "missing or malformed Authorization header"}})
			return
		}
		token := strings.TrimPrefix(header, prefix)

		if subtle.ConstantTimeCompare([]byte(token), []byte(cfg.APIKey)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "invalid API key"}})
			return
		}

		c.Next()
	}
}
