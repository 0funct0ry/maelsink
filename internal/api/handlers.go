package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/0funct0ry/maelsink/internal/store"
	"github.com/0funct0ry/maelsink/internal/version"
)

const (
	defaultLimit = 50
	maxLimit     = 500
)

type handlers struct {
	store store.MessageStore
}

// messageSummary is the Message (summary) shape from SPEC.md §5.1.
type messageSummary struct {
	ID              string   `json:"id"`
	From            string   `json:"from"`
	To              []string `json:"to"`
	Cc              []string `json:"cc"`
	Subject         string   `json:"subject"`
	SizeBytes       int64    `json:"size_bytes"`
	HasAttachments  bool     `json:"has_attachments"`
	AttachmentCount int      `json:"attachment_count"`
	ReceivedAt      string   `json:"received_at"`
	ParseWarning    bool     `json:"parse_warning"`
}

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

func addrStrings(addrs []store.Address) []string {
	out := make([]string, len(addrs))
	for i, a := range addrs {
		out[i] = a.Address
	}
	return out
}

func toSummary(msg *store.Message) messageSummary {
	from := ""
	if len(msg.From) > 0 {
		from = msg.From[0].Address
	}
	return messageSummary{
		ID:              msg.ID,
		From:            from,
		To:              addrStrings(msg.To),
		Cc:              addrStrings(msg.Cc),
		Subject:         msg.Subject,
		SizeBytes:       msg.Size,
		HasAttachments:  msg.AttachmentCount > 0,
		AttachmentCount: msg.AttachmentCount,
		ReceivedAt:      msg.ReceivedAt.UTC().Format(time.RFC3339),
		ParseWarning:    msg.ParseWarning,
	}
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
		Sort:    c.DefaultQuery("sort", store.SortReceivedAtDesc),
	}
	if filter.Sort != store.SortReceivedAtDesc && filter.Sort != store.SortReceivedAtAsc {
		return filter, errors.New("sort must be one of received_at_desc, received_at_asc")
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

func (h *handlers) listMessages(c *gin.Context) {
	filter, err := parseListFilter(c)
	if err != nil {
		respondValidation(c, err.Error())
		return
	}

	msgs, total, err := h.store.List(c.Request.Context(), filter)
	if err != nil {
		respondInternal(c, err.Error())
		return
	}

	summaries := make([]messageSummary, len(msgs))
	for i, m := range msgs {
		summaries[i] = toSummary(m)
	}

	c.JSON(http.StatusOK, listResponse{Messages: summaries, Total: total, Limit: filter.Limit, Offset: filter.Offset})
}

func (h *handlers) getMessage(c *gin.Context) {
	id := c.Param("id")
	msg, err := h.store.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondNotFound(c, id)
			return
		}
		respondInternal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, toDetail(msg))
}

func (h *handlers) deleteMessage(c *gin.Context) {
	id := c.Param("id")
	if err := h.store.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondNotFound(c, id)
			return
		}
		respondInternal(c, err.Error())
		return
	}
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
	c.Status(http.StatusNoContent)
}

func (h *handlers) rawMessage(c *gin.Context) {
	id := c.Param("id")
	msg, err := h.store.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondNotFound(c, id)
			return
		}
		respondInternal(c, err.Error())
		return
	}
	c.Data(http.StatusOK, "message/rfc822", msg.RawSource)
}

func (h *handlers) exportMessage(c *gin.Context) {
	id := c.Param("id")
	msg, err := h.store.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondNotFound(c, id)
			return
		}
		respondInternal(c, err.Error())
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+id+`.eml"`)
	c.Data(http.StatusOK, "message/rfc822", msg.RawSource)
}

func (h *handlers) getAttachment(c *gin.Context) {
	id, attID := c.Param("id"), c.Param("attId")
	msg, err := h.store.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondNotFound(c, id)
			return
		}
		respondInternal(c, err.Error())
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
		"total_messages":   s.TotalMessages,
		"total_size_bytes": s.TotalSizeBytes,
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
