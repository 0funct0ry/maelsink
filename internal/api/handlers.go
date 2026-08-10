package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/0funct0ry/maelsink/internal/events"
	"github.com/0funct0ry/maelsink/internal/store"
	"github.com/0funct0ry/maelsink/internal/version"
)

const (
	defaultLimit = 50
	maxLimit     = 500
)

// invalidQueryMessage is shown to the user for a store.ErrInvalidQuery,
// deliberately generic — the underlying SQLite/FTS5 error text (column
// names, "SQL logic error", etc.) is an implementation detail that must
// never reach the client. See internal/store's ErrInvalidQuery doc comment.
const invalidQueryMessage = "Invalid search query. Check your quotes, boolean operators (AND/OR/NOT), and column filters (e.g. subject:...)."

type handlers struct {
	store store.MessageStore
	bus   *events.Bus
}

// messageSummary is the Message (summary) shape from SPEC.md §5.1. It also
// backs the message.created WebSocket event payload (SPEC.md §5.5), so the
// type itself lives in internal/store (see store.MessageSummary) where both
// internal/api and internal/smtp can build it without importing each other.
type messageSummary = store.MessageSummary

// attachmentSummary is a single entry in messageDetail.Attachments.
type attachmentSummary struct {
	ID          string  `json:"id"`
	Filename    string  `json:"filename"`
	ContentType string  `json:"content_type"`
	SizeBytes   int64   `json:"size_bytes"`
	ContentID   *string `json:"content_id"`
}

// headerJSON is a single entry in messageDetail.Headers.
type headerJSON struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type messageDetail struct {
	messageSummary
	Headers      []headerJSON        `json:"headers"`
	TextBody     string              `json:"text_body"`
	HTMLBody     string              `json:"html_body"`
	Attachments  []attachmentSummary `json:"attachments"`
	RawSizeBytes int64               `json:"raw_size_bytes"`
}

// toSummary builds the summary JSON shape for msg.
func toSummary(msg *store.Message) messageSummary {
	return store.NewMessageSummary(msg)
}

func toDetail(msg *store.Message) messageDetail {
	headers := make([]headerJSON, len(msg.Headers))
	for i, h := range msg.Headers {
		headers[i] = headerJSON{Name: h.Name, Value: h.Value}
	}

	attachments := make([]attachmentSummary, 0, len(msg.Attachments)+len(msg.InlineImages))
	for _, a := range msg.Attachments {
		attachments = append(attachments, attachmentSummary{
			ID: a.ID, Filename: a.Filename, ContentType: a.ContentType, SizeBytes: a.Size,
		})
	}
	for _, img := range msg.InlineImages {
		cid := img.ContentID
		attachments = append(attachments, attachmentSummary{
			ID: img.ID, Filename: img.Filename, ContentType: img.ContentType, SizeBytes: img.Size, ContentID: &cid,
		})
	}

	return messageDetail{
		messageSummary: toSummary(msg),
		Headers:        headers,
		TextBody:       msg.TextBody,
		HTMLBody:       msg.HTMLBody,
		Attachments:    attachments,
		RawSizeBytes:   int64(len(msg.RawSource)),
	}
}

// listResponse wraps the summary array with the pagination metadata callers
// (CLI table view, Web UI pager) need alongside it.
type listResponse struct {
	Messages []messageSummary `json:"messages"`
	Total    int              `json:"total"`
	Limit    int              `json:"limit"`
	Offset   int              `json:"offset"`
}

func parseListFilter(c *gin.Context) (store.ListFilter, error) {
	filter := store.ListFilter{
		Query:   c.Query("q"),
		From:    c.Query("from"),
		To:      c.Query("to"),
		Subject: c.Query("subject"),
		Cc:      c.Query("cc"),
		Bcc:     c.Query("bcc"),
		Sort:    c.DefaultQuery("sort", store.SortReceivedAtDesc),
		Tag:     c.Query("tag"),
	}
	if filter.Sort != store.SortReceivedAtDesc && filter.Sort != store.SortReceivedAtAsc {
		return filter, errors.New("sort must be one of received_at_desc, received_at_asc")
	}

	var err error
	if filter.Read, err = parseOptionalBoolQuery(c, "read"); err != nil {
		return filter, err
	}
	if filter.HasAttachments, err = parseOptionalBoolQuery(c, "has_attachments"); err != nil {
		return filter, err
	}
	if filter.ParseWarning, err = parseOptionalBoolQuery(c, "parse_warning"); err != nil {
		return filter, err
	}

	filter.Limit = defaultLimit
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return filter, errors.New("limit must be a non-negative integer")
		}
		filter.Limit = n
	}
	if filter.Limit > maxLimit {
		filter.Limit = maxLimit
	}

	if v := c.Query("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return filter, errors.New("offset must be a non-negative integer")
		}
		filter.Offset = n
	}

	if v := c.Query("since"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return filter, errors.New("since must be an RFC3339 timestamp")
		}
		filter.Since = t
	}
	if v := c.Query("until"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return filter, errors.New("until must be an RFC3339 timestamp")
		}
		filter.Until = t
	}

	return filter, nil
}

// parseOptionalBoolQuery returns nil (unset) when the query param is
// absent, else a pointer to its parsed bool value.
func parseOptionalBoolQuery(c *gin.Context, name string) (*bool, error) {
	v := c.Query(name)
	if v == "" {
		return nil, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return nil, errors.New(name + " must be a boolean")
	}
	return &b, nil
}

func (h *handlers) listMessages(c *gin.Context) {
	filter, err := parseListFilter(c)
	if err != nil {
		respondValidation(c, err.Error())
		return
	}

	msgs, total, err := h.store.List(c.Request.Context(), filter)
	if err != nil {
		if errors.Is(err, store.ErrInvalidQuery) {
			respondError(c, http.StatusBadRequest, "invalid_query", invalidQueryMessage)
			return
		}
		respondInternal(c, err.Error())
		return
	}

	summaries := make([]messageSummary, len(msgs))
	for i, m := range msgs {
		summaries[i] = toSummary(m)
	}

	c.JSON(http.StatusOK, listResponse{Messages: summaries, Total: total, Limit: filter.Limit, Offset: filter.Offset})
}

// handleStoreErr classifies an error from a by-id/by-prefix store lookup
// into the matching error response: not found, an ambiguous short prefix
// (SPEC.md §5.2's ID-prefix resolution), or a genuine internal error.
func handleStoreErr(c *gin.Context, id string, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		respondNotFound(c, id)
	case errors.Is(err, store.ErrAmbiguousID):
		respondAmbiguousID(c, id)
	default:
		respondInternal(c, err.Error())
	}
}

func (h *handlers) getMessage(c *gin.Context) {
	id := c.Param("id")
	msg, err := h.store.Get(c.Request.Context(), id)
	if err != nil {
		handleStoreErr(c, id, err)
		return
	}
	c.JSON(http.StatusOK, toDetail(msg))
}

// markReadBody is an optional JSON body for PATCH .../read: {"read": false}
// marks the message unread; an absent/empty body (or {"read": true}) marks
// it read, preserving the endpoint's original mark-as-read-only behavior.
type markReadBody struct {
	Read *bool `json:"read"`
}

func (h *handlers) markRead(c *gin.Context) {
	id := c.Param("id")

	read := true
	var body markReadBody
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&body); err != nil {
			respondError(c, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if body.Read != nil {
			read = *body.Read
		}
	}

	if err := h.store.MarkRead(c.Request.Context(), id, read); err != nil {
		handleStoreErr(c, id, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *handlers) deleteMessage(c *gin.Context) {
	id := c.Param("id")

	// id may be an unambiguous prefix (per MessageStore.Delete); resolve it
	// to the full ID before publishing message.deleted, so WS clients
	// matching on payload.id always see the canonical ID.
	msg, err := h.store.Get(c.Request.Context(), id)
	if err != nil {
		handleStoreErr(c, id, err)
		return
	}

	if err := h.store.Delete(c.Request.Context(), id); err != nil {
		handleStoreErr(c, id, err)
		return
	}
	h.bus.Publish(events.MessageDeleted(msg.ID))
	c.Status(http.StatusNoContent)
}

func (h *handlers) clearMessages(c *gin.Context) {
	if c.Query("confirm") != "true" {
		respondError(c, http.StatusBadRequest, "confirmation_required", "bulk delete requires ?confirm=true")
		return
	}
	if err := h.store.Clear(c.Request.Context()); err != nil {
		respondInternal(c, err.Error())
		return
	}
	h.bus.Publish(events.MessagesCleared())
	c.Status(http.StatusNoContent)
}

func (h *handlers) rawMessage(c *gin.Context) {
	id := c.Param("id")
	msg, err := h.store.Get(c.Request.Context(), id)
	if err != nil {
		handleStoreErr(c, id, err)
		return
	}
	c.Data(http.StatusOK, "message/rfc822", msg.RawSource)
}

func (h *handlers) exportMessage(c *gin.Context) {
	id := c.Param("id")
	msg, err := h.store.Get(c.Request.Context(), id)
	if err != nil {
		handleStoreErr(c, id, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+msg.ID+`.eml"`)
	c.Data(http.StatusOK, "message/rfc822", msg.RawSource)
}

func (h *handlers) getAttachment(c *gin.Context) {
	id, attID := c.Param("id"), c.Param("attId")
	msg, err := h.store.Get(c.Request.Context(), id)
	if err != nil {
		handleStoreErr(c, id, err)
		return
	}

	for _, a := range msg.Attachments {
		if a.ID == attID {
			c.Header("Content-Disposition", `attachment; filename="`+a.Filename+`"`)
			c.Data(http.StatusOK, a.ContentType, a.Data)
			return
		}
	}
	for _, img := range msg.InlineImages {
		if img.ID == attID {
			c.Header("Content-Disposition", `attachment; filename="`+img.Filename+`"`)
			c.Data(http.StatusOK, img.ContentType, img.Data)
			return
		}
	}

	respondError(c, http.StatusNotFound, "attachment_not_found", "no attachment with id "+attID+" on message "+id)
}

func (h *handlers) stats(c *gin.Context) {
	s, err := h.store.Stats(c.Request.Context())
	if err != nil {
		respondInternal(c, err.Error())
		return
	}

	resp := gin.H{
		"total_messages":      s.TotalMessages,
		"total_size_bytes":    s.TotalSizeBytes,
		"unread_count":        s.UnreadCount,
		"attachment_count":    s.AttachmentCount,
		"parse_warning_count": s.ParseWarningCount,
	}
	if s.OldestReceivedAt != nil {
		resp["oldest_received_at"] = s.OldestReceivedAt.UTC().Format(time.RFC3339)
	} else {
		resp["oldest_received_at"] = nil
	}
	if s.NewestReceivedAt != nil {
		resp["newest_received_at"] = s.NewestReceivedAt.UTC().Format(time.RFC3339)
	} else {
		resp["newest_received_at"] = nil
	}
	c.JSON(http.StatusOK, resp)
}

// tagCountJSON is one row of GET /api/v1/tags.
type tagCountJSON struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

func (h *handlers) listTags(c *gin.Context) {
	tags, err := h.store.Tags(c.Request.Context())
	if err != nil {
		respondInternal(c, err.Error())
		return
	}
	out := make([]tagCountJSON, len(tags))
	for i, t := range tags {
		out[i] = tagCountJSON{Tag: t.Tag, Count: t.Count}
	}
	c.JSON(http.StatusOK, out)
}

func (h *handlers) health(c *gin.Context) {
	dbStatus := "ok"
	overall := "ok"
	if err := h.store.Ping(c.Request.Context()); err != nil {
		dbStatus = "error"
		overall = "degraded"
	}

	status := http.StatusOK
	if overall != "ok" {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"status": overall, "db": dbStatus, "smtp": "listening"})
}

func (h *handlers) version(c *gin.Context) {
	c.JSON(http.StatusOK, version.Get())
}
