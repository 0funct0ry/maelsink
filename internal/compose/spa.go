package compose

import (
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

// registerSPARoutes serves static assets byte-for-byte from assets, and
// falls back to indexHTML for every other GET so the SPA's client-side
// router can take over. Unlike internal/webui, compose has no
// BasePath/X-Forwarded-Prefix story in this milestone — it runs standalone,
// not behind a reverse-proxy subpath.
func registerSPARoutes(engine *gin.Engine, assets fs.FS, indexHTML []byte) {
	fileServer := http.FileServer(http.FS(assets))

	serveIndex := func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	}

	engine.GET("/", serveIndex)
	engine.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet {
			c.Status(http.StatusNotFound)
			return
		}

		reqPath := c.Request.URL.Path
		if len(reqPath) > 0 && reqPath[0] == '/' {
			reqPath = reqPath[1:]
		}

		if reqPath != "" {
			if f, err := assets.Open(reqPath); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}

		// Not a real asset: an SPA client-side route (e.g. /messages/abc) or
		// the bare root — serve index.html so the frontend router takes over
		// instead of a 404.
		serveIndex(c)
	})
}
