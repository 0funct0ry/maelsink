package webui

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/0funct0ry/maelsink/internal/webauth"
)

// basicAuthMiddleware gates every request on this router with Basic Auth
// against authFile, except the /api/v1 subtree mounted read-through from
// internal/api — that surface continues to rely solely on api.auth's bearer
// key (SPEC.md §5.4). Static assets, /ui-api/*, and /ws are all gated
// uniformly so the browser's native Basic Auth dialog protects the whole
// human-facing surface on first load.
func basicAuthMiddleware(authFile, basePath string) gin.HandlerFunc {
	exempt := basePath + "/api/v1"
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, exempt) {
			c.Next()
			return
		}

		user, pass, ok := c.Request.BasicAuth()
		if !ok || !webauth.Verify(authFile, user, pass) {
			c.Header("WWW-Authenticate", `Basic realm="maelsink"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}
