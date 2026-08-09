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

func respondValidation(c *gin.Context, message string) {
	respondError(c, 400, "validation_error", message)
}

func respondInternal(c *gin.Context, message string) {
	respondError(c, 500, "internal_error", message)
}
