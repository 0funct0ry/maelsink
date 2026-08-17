package compose

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/version"
)

func statsHandler(client *cliclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats, err := client.Stats(c.Request.Context())
		if err != nil {
			writeClientError(c, err)
			return
		}
		c.JSON(http.StatusOK, stats)
	}
}

// versionResponse reports both the target's version (proxied) and
// compose's own build version, per SPEC.md §7.7.4.2.
type versionResponse struct {
	Target  *cliclient.VersionInfo `json:"target"`
	Compose version.Info           `json:"compose"`
}

func versionHandler(client *cliclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		target, err := client.Version(c.Request.Context())
		if err != nil {
			writeClientError(c, err)
			return
		}
		c.JSON(http.StatusOK, versionResponse{Target: target, Compose: version.Get()})
	}
}

// exportHandler proxies the target's bulk export equivalent (filtered by
// the same query params as list), streaming the resulting .zip back with a
// Content-Disposition header so the browser triggers a real file download
// — there is no local output directory in a browser (SPEC.md §7.7.4.2).
func exportHandler(client *cliclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := c.Request.URL.Query()
		params := cliclient.ListParams{
			Query:   q.Get("q"),
			From:    q.Get("from"),
			To:      q.Get("to"),
			Subject: q.Get("subject"),
			Since:   q.Get("since"),
			Until:   q.Get("until"),
			Sort:    q.Get("sort"),
		}

		data, err := client.BulkExport(c.Request.Context(), params)
		if err != nil {
			writeClientError(c, err)
			return
		}
		c.Header("Content-Disposition", `attachment; filename="export.zip"`)
		c.Data(http.StatusOK, "application/zip", data)
	}
}

// attachmentHandler proxies a single attachment's raw bytes, streamed back
// with a Content-Disposition header (same approach as exportHandler) so the
// browser saves a real, openable file.
func attachmentHandler(client *cliclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, contentType, filename, err := client.Attachment(c.Request.Context(), c.Param("id"), c.Param("attachmentId"))
		if err != nil {
			writeClientError(c, err)
			return
		}
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		c.Header("Content-Disposition", "attachment; filename="+strconv.Quote(filename))
		c.Data(http.StatusOK, contentType, data)
	}
}
