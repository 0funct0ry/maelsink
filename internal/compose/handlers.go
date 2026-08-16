package compose

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/0funct0ry/maelsink/internal/cliclient"
)

// writeClientError classifies err — a *cliclient.HTTPError (target
// reachable but rejected the request, forwarded verbatim) vs. a plain
// transport error (target unreachable, 502) — and writes the matching
// response. Errors are never swallowed (SPEC.md §7.7.5).
func writeClientError(c *gin.Context, err error) {
	if httpErr, ok := err.(*cliclient.HTTPError); ok {
		c.JSON(httpErr.Status, gin.H{"error": gin.H{"code": httpErr.Code, "message": httpErr.Message}})
		return
	}
	c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "target_unreachable", "message": err.Error()}})
}

func listMessagesHandler(client *cliclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := c.Request.URL.Query()
		limit, _ := strconv.Atoi(q.Get("limit"))
		offset, _ := strconv.Atoi(q.Get("offset"))
		params := cliclient.ListParams{
			Query:   q.Get("q"),
			From:    q.Get("from"),
			To:      q.Get("to"),
			Subject: q.Get("subject"),
			Cc:      q.Get("cc"),
			Bcc:     q.Get("bcc"),
			Limit:   limit,
			Offset:  offset,
			Since:   q.Get("since"),
			Until:   q.Get("until"),
			Sort:    q.Get("sort"),
		}

		result, err := client.List(c.Request.Context(), params)
		if err != nil {
			writeClientError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func getMessageHandler(client *cliclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		msg, err := client.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			writeClientError(c, err)
			return
		}
		c.JSON(http.StatusOK, msg)
	}
}

func deleteMessageHandler(client *cliclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := client.Delete(c.Request.Context(), c.Param("id")); err != nil {
			writeClientError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func clearMessagesHandler(client *cliclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := client.Clear(c.Request.Context()); err != nil {
			writeClientError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// healthResponse is compose's own health envelope — not the target's raw
// body — so the frontend's connection-status indicator can stay dumb.
type healthResponse struct {
	TargetReachable bool   `json:"target_reachable"`
	Status          string `json:"status"` // "green" | "yellow" | "red"
	TargetHealth    any    `json:"target_health,omitempty"`
	Error           string `json:"error,omitempty"`
}

// healthHandler thinly proxies GET /api/v1/health on the target. It always
// responds 200 regardless of target state — compose's own endpoint IS
// reachable even when the target isn't, and the frontend polls this
// endpoint to drive its red/yellow/green indicator, so a 5xx here would
// break the poller's own success path.
func healthHandler(client *cliclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.BaseURL+"/api/v1/health", nil)
		if err != nil {
			c.JSON(http.StatusOK, healthResponse{Status: "red", Error: err.Error()})
			return
		}

		httpClient := client.HTTPClient
		if httpClient == nil {
			httpClient = http.DefaultClient
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			c.JSON(http.StatusOK, healthResponse{TargetReachable: false, Status: "red", Error: err.Error()})
			return
		}
		defer resp.Body.Close()

		var body any
		_ = json.NewDecoder(resp.Body).Decode(&body)

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			c.JSON(http.StatusOK, healthResponse{TargetReachable: true, Status: "green", TargetHealth: body})
			return
		}
		c.JSON(http.StatusOK, healthResponse{TargetReachable: true, Status: "yellow", TargetHealth: body})
	}
}
