package api

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

// errorEnvelope is the standard error shape from SPEC.md §5.3:
//
//	{ "error": { "code": "message_not_found", "message": "no message with id msg_xxx" } }
type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, errorEnvelope{Error: errorBody{Code: code, Message: message}})
}

func respondNotFound(c *gin.Context, id string) {
	respondError(c, 404, "message_not_found", "no message with id "+id)
}

func respondAmbiguousID(c *gin.Context, id string) {
	respondError(c, 400, "ambiguous_id", "id prefix "+id+" matches more than one message")
}

func respondSessionNotFound(c *gin.Context, id string) {
	respondError(c, 404, "session_not_found", "no session with id "+id)
}

func respondSessionAmbiguousID(c *gin.Context, id string) {
	respondError(c, 400, "ambiguous_id", "id prefix "+id+" matches more than one session")
}

func respondValidation(c *gin.Context, message string) {
	respondError(c, 400, "validation_error", message)
}

func respondInternal(c *gin.Context, message string) {
	respondError(c, 500, "internal_error", message)
}

// respondInternalErr logs the real error server-side (never the client's
// business) and responds with a generic, stable message — the raw
// SQLite/driver/OS error string is an implementation detail that must never
// reach the client, mirroring recoveryHandler's existing panic-handling
// pattern (internal/api/middleware.go).
func respondInternalErr(c *gin.Context, logger *slog.Logger, err error) {
	logger.Error("internal error", "error", err, "path", c.Request.URL.Path)
	respondInternal(c, "an internal error occurred")
}

func respondTagNotFound(c *gin.Context, name string) {
	respondError(c, 404, "tag_not_found", "no tag named "+name)
}

func respondTagExists(c *gin.Context, name string) {
	respondError(c, 409, "tag_exists", "tag "+name+" already exists")
}
