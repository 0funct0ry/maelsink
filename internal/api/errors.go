package api

import "github.com/gin-gonic/gin"

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

func respondTagNotFound(c *gin.Context, name string) {
	respondError(c, 404, "tag_not_found", "no tag named "+name)
}

func respondTagExists(c *gin.Context, name string) {
	respondError(c, 409, "tag_exists", "tag "+name+" already exists")
}
