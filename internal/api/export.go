package api

import (
	"archive/zip"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/0funct0ry/maelsink/internal/store"
)

// bulkExport handles GET /api/v1/messages/export: the same filters as
// listMessages (minus pagination) select the match set, which is streamed
// back as a .zip of <id>.eml files (SPEC.md §5.2).
func (h *handlers) bulkExport(c *gin.Context) {
	filter, err := parseListFilter(c)
	if err != nil {
		respondValidation(c, err.Error())
		return
	}
	filter.Limit = 0
	filter.Offset = 0

	summaries, _, err := h.store.List(c.Request.Context(), filter)
	if err != nil {
		respondInternal(c, err.Error())
		return
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", `attachment; filename="messages.zip"`)
	c.Status(http.StatusOK)

	zw := zip.NewWriter(c.Writer)
	defer zw.Close()

	for _, summary := range summaries {
		msg, err := h.store.Get(c.Request.Context(), summary.ID)
		if err != nil {
			if err == store.ErrNotFound {
				continue
			}
			return
		}
		f, err := zw.Create(msg.ID + ".eml")
		if err != nil {
			return
		}
		if _, err := f.Write(msg.RawSource); err != nil {
			return
		}
	}
}
