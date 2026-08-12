package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
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
	store  store.MessageStore
	bus    *events.Bus
	logger *slog.Logger
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
	// SessionID links to the SMTP session that produced this message
	// (M8.4), for the Message Detail -> Session Detail cross-link. Omitted
	// for messages saved outside a tracked SMTP session.
	SessionID string `json:"session_id,omitempty"`
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
		SessionID:      msg.SessionID,
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
		Tags:    c.QueryArray("tag"),
		TagMode: c.DefaultQuery("tag_mode", "any"),
	}
	if filter.Sort != store.SortReceivedAtDesc && filter.Sort != store.SortReceivedAtAsc {
		return filter, errors.New("sort must be one of received_at_desc, received_at_asc")
	}
	if filter.TagMode != "any" && filter.TagMode != "all" {
		return filter, errors.New("tag_mode must be one of any, all")
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
		respondInternalErr(c, h.logger, err)
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
func (h *handlers) handleStoreErr(c *gin.Context, id string, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		respondNotFound(c, id)
	case errors.Is(err, store.ErrAmbiguousID):
		respondAmbiguousID(c, id)
	default:
		respondInternalErr(c, h.logger, err)
	}
}

func (h *handlers) getMessage(c *gin.Context) {
	id := c.Param("id")
	msg, err := h.store.Get(c.Request.Context(), id)
	if err != nil {
		h.handleStoreErr(c, id, err)
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
		h.handleStoreErr(c, id, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// updateTagsBody is the JSON body for PATCH .../tags: {"add": [...], "remove": [...]}.
type updateTagsBody struct {
	Add    []string `json:"add"`
	Remove []string `json:"remove"`
}

func (h *handlers) updateTags(c *gin.Context) {
	id := c.Param("id")

	var body updateTagsBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	// Resolve the id to the canonical ID up front (per MessageStore.Get),
	// so a short prefix only needs resolving once and every mutation below
	// (and the published event) uses the full ID.
	msg, err := h.store.Get(c.Request.Context(), id)
	if err != nil {
		h.handleStoreErr(c, id, err)
		return
	}

	for _, t := range body.Remove {
		if err := h.store.RemoveTag(c.Request.Context(), msg.ID, t); err != nil {
			if errors.Is(err, store.ErrInvalidTag) {
				respondValidation(c, err.Error())
				return
			}
			h.handleStoreErr(c, id, err)
			return
		}
	}
	for _, t := range body.Add {
		if err := h.store.AddTag(c.Request.Context(), msg.ID, t); err != nil {
			if errors.Is(err, store.ErrInvalidTag) {
				respondValidation(c, err.Error())
				return
			}
			h.handleStoreErr(c, id, err)
			return
		}
	}

	updated, err := h.store.Get(c.Request.Context(), msg.ID)
	if err != nil {
		h.handleStoreErr(c, id, err)
		return
	}
	h.bus.Publish(events.MessageTagsUpdated(updated.ID, updated.Tags))
	c.JSON(http.StatusOK, toSummary(updated))
}

func (h *handlers) deleteMessage(c *gin.Context) {
	id := c.Param("id")

	// id may be an unambiguous prefix (per MessageStore.Delete); resolve it
	// to the full ID before publishing message.deleted, so WS clients
	// matching on payload.id always see the canonical ID.
	msg, err := h.store.Get(c.Request.Context(), id)
	if err != nil {
		h.handleStoreErr(c, id, err)
		return
	}

	if err := h.store.Delete(c.Request.Context(), id); err != nil {
		h.handleStoreErr(c, id, err)
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
		respondInternalErr(c, h.logger, err)
		return
	}
	h.bus.Publish(events.MessagesCleared())
	c.Status(http.StatusNoContent)
}

func (h *handlers) rawMessage(c *gin.Context) {
	id := c.Param("id")
	msg, err := h.store.Get(c.Request.Context(), id)
	if err != nil {
		h.handleStoreErr(c, id, err)
		return
	}
	c.Data(http.StatusOK, "message/rfc822", msg.RawSource)
}

func (h *handlers) exportMessage(c *gin.Context) {
	id := c.Param("id")
	msg, err := h.store.Get(c.Request.Context(), id)
	if err != nil {
		h.handleStoreErr(c, id, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+msg.ID+`.eml"`)
	c.Data(http.StatusOK, "message/rfc822", msg.RawSource)
}

func (h *handlers) getAttachment(c *gin.Context) {
	id, attID := c.Param("id"), c.Param("attId")
	msg, err := h.store.Get(c.Request.Context(), id)
	if err != nil {
		h.handleStoreErr(c, id, err)
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
		respondInternalErr(c, h.logger, err)
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

// tagStatsJSON is one row of GET /api/v1/tags.
type tagStatsJSON struct {
	Name     string  `json:"name"`
	Color    string  `json:"color"`
	Count    int     `json:"count"`
	LastUsed *string `json:"last_used"`
}

func toTagStatsJSON(t store.TagStats) tagStatsJSON {
	out := tagStatsJSON{Name: t.Name, Color: t.Color, Count: t.Count}
	if t.LastUsed != nil {
		s := t.LastUsed.UTC().Format(time.RFC3339)
		out.LastUsed = &s
	}
	return out
}

func (h *handlers) listTags(c *gin.Context) {
	tags, err := h.store.ListTagsWithStats(c.Request.Context())
	if err != nil {
		respondInternalErr(c, h.logger, err)
		return
	}
	out := make([]tagStatsJSON, len(tags))
	for i, t := range tags {
		out[i] = toTagStatsJSON(t)
	}
	c.JSON(http.StatusOK, out)
}

// handleTagStoreErr classifies an error from a tag mutation into the
// matching error response.
func (h *handlers) handleTagStoreErr(c *gin.Context, name string, err error) {
	switch {
	case errors.Is(err, store.ErrTagNotFound):
		respondTagNotFound(c, name)
	case errors.Is(err, store.ErrTagExists):
		respondTagExists(c, name)
	case errors.Is(err, store.ErrInvalidTag):
		respondValidation(c, err.Error())
	default:
		respondInternalErr(c, h.logger, err)
	}
}

// findTagStats returns the TagStats row for name, or nil if it doesn't
// exist.
func (h *handlers) findTagStats(c *gin.Context, name string) (*store.TagStats, error) {
	tags, err := h.store.ListTagsWithStats(c.Request.Context())
	if err != nil {
		return nil, err
	}
	for _, t := range tags {
		if t.Name == name {
			return &t, nil
		}
	}
	return nil, nil
}

// createTagBody is the JSON body for POST /api/v1/tags.
type createTagBody struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

func (h *handlers) createTag(c *gin.Context) {
	var body createTagBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	if strings.TrimSpace(body.Name) == "" || strings.TrimSpace(body.Color) == "" {
		respondValidation(c, "name and color are both required")
		return
	}

	if err := h.store.CreateTag(c.Request.Context(), body.Name, body.Color); err != nil {
		h.handleTagStoreErr(c, body.Name, err)
		return
	}
	h.bus.Publish(events.TagCreated(body.Name, body.Color))

	ts, err := h.findTagStats(c, strings.TrimSpace(body.Name))
	if err != nil {
		respondInternalErr(c, h.logger, err)
		return
	}
	c.JSON(http.StatusCreated, toTagStatsJSON(*ts))
}

// patchTagBody is the JSON body for PATCH /api/v1/tags/:name.
type patchTagBody struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

func (h *handlers) patchTag(c *gin.Context) {
	name := c.Param("name")

	var body patchTagBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	if (body.Name == nil || strings.TrimSpace(*body.Name) == "") && (body.Color == nil || strings.TrimSpace(*body.Color) == "") {
		respondValidation(c, "at least one of name or color is required")
		return
	}

	effectiveName := name

	if body.Name != nil && strings.TrimSpace(*body.Name) != "" && strings.TrimSpace(*body.Name) != name {
		newName := strings.TrimSpace(*body.Name)
		existing, err := h.findTagStats(c, newName)
		if err != nil {
			respondInternalErr(c, h.logger, err)
			return
		}
		merged := existing != nil

		if err := h.store.RenameTag(c.Request.Context(), name, newName); err != nil {
			h.handleTagStoreErr(c, name, err)
			return
		}
		h.bus.Publish(events.TagRenamed(name, newName, merged))
		effectiveName = newName
	}

	if body.Color != nil && strings.TrimSpace(*body.Color) != "" {
		color := strings.TrimSpace(*body.Color)
		if err := h.store.RecolorTag(c.Request.Context(), effectiveName, color); err != nil {
			h.handleTagStoreErr(c, effectiveName, err)
			return
		}
		h.bus.Publish(events.TagRecolored(effectiveName, color))
	}

	ts, err := h.findTagStats(c, effectiveName)
	if err != nil {
		respondInternalErr(c, h.logger, err)
		return
	}
	if ts == nil {
		respondTagNotFound(c, effectiveName)
		return
	}
	c.JSON(http.StatusOK, toTagStatsJSON(*ts))
}

func (h *handlers) deleteTag(c *gin.Context) {
	name := c.Param("name")
	if err := h.store.DeleteTag(c.Request.Context(), name); err != nil {
		h.handleTagStoreErr(c, name, err)
		return
	}
	h.bus.Publish(events.TagDeleted(name))
	c.Status(http.StatusNoContent)
}

func (h *handlers) deleteTagWithMessages(c *gin.Context) {
	name := c.Param("name")

	msgs, _, err := h.store.List(c.Request.Context(), store.ListFilter{Tag: name})
	if err != nil {
		respondInternalErr(c, h.logger, err)
		return
	}

	if err := h.store.DeleteTagWithMessages(c.Request.Context(), name); err != nil {
		h.handleTagStoreErr(c, name, err)
		return
	}

	for _, msg := range msgs {
		h.bus.Publish(events.MessageDeleted(msg.ID))
	}
	h.bus.Publish(events.TagDeleted(name))
	c.Status(http.StatusNoContent)
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

// sessionSummaryJSON is the Session (summary) shape from SPEC.md §5.1/§5.5
// (M8.4) — no transcript, used by both GET /api/v1/sessions and the
// session.started/session.completed WS event payloads.
type sessionSummaryJSON struct {
	ID         string  `json:"id"`
	ClientIP   string  `json:"client_ip"`
	ClientHELO string  `json:"client_helo"`
	StartedAt  string  `json:"started_at"`
	EndedAt    *string `json:"ended_at"`
	Status     string  `json:"status"`
	MessageID  *string `json:"message_id"`
}

// transcriptLineJSON is a single entry in sessionDetail.Transcript.
type transcriptLineJSON struct {
	Direction string `json:"direction"`
	Line      string `json:"line"`
	Position  int    `json:"position"`
}

type sessionDetailJSON struct {
	sessionSummaryJSON
	Transcript []transcriptLineJSON `json:"transcript"`
}

func toSessionSummary(sess *store.SessionSummary) sessionSummaryJSON {
	var endedAt *string
	if sess.EndedAt != nil {
		s := sess.EndedAt.UTC().Format(time.RFC3339)
		endedAt = &s
	}
	return sessionSummaryJSON{
		ID:         sess.ID,
		ClientIP:   sess.ClientIP,
		ClientHELO: sess.ClientHELO,
		StartedAt:  sess.StartedAt.UTC().Format(time.RFC3339),
		EndedAt:    endedAt,
		Status:     sess.Status,
		MessageID:  sess.MessageID,
	}
}

func toSessionDetail(sess *store.Session) sessionDetailJSON {
	summary := toSessionSummary(&store.SessionSummary{
		ID: sess.ID, ClientIP: sess.ClientIP, ClientHELO: sess.ClientHELO,
		StartedAt: sess.StartedAt, EndedAt: sess.EndedAt, Status: sess.Status, MessageID: sess.MessageID,
	})
	lines := make([]transcriptLineJSON, len(sess.Transcript))
	for i, l := range sess.Transcript {
		lines[i] = transcriptLineJSON{Direction: string(l.Direction), Line: l.Line, Position: l.Position}
	}
	return sessionDetailJSON{sessionSummaryJSON: summary, Transcript: lines}
}

// listSessionsResponse wraps the summary array with pagination metadata,
// mirroring listResponse.
type listSessionsResponse struct {
	Sessions []sessionSummaryJSON `json:"sessions"`
	Total    int                  `json:"total"`
	Limit    int                  `json:"limit"`
	Offset   int                  `json:"offset"`
}

func parseSessionListFilter(c *gin.Context) (store.SessionListFilter, error) {
	filter := store.SessionListFilter{
		Status:   c.Query("status"),
		ClientIP: c.Query("client_ip"),
		Sort:     c.DefaultQuery("sort", store.SortStartedAtDesc),
	}
	if filter.Sort != store.SortStartedAtDesc && filter.Sort != store.SortStartedAtAsc {
		return filter, errors.New("sort must be one of started_at_desc, started_at_asc")
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

func (h *handlers) listSessions(c *gin.Context) {
	filter, err := parseSessionListFilter(c)
	if err != nil {
		respondValidation(c, err.Error())
		return
	}

	sessions, total, err := h.store.ListSessions(c.Request.Context(), filter)
	if err != nil {
		respondInternalErr(c, h.logger, err)
		return
	}

	summaries := make([]sessionSummaryJSON, len(sessions))
	for i, sess := range sessions {
		summaries[i] = toSessionSummary(sess)
	}

	c.JSON(http.StatusOK, listSessionsResponse{Sessions: summaries, Total: total, Limit: filter.Limit, Offset: filter.Offset})
}

func (h *handlers) getSession(c *gin.Context) {
	id := c.Param("id")
	sess, err := h.store.GetSession(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			respondSessionNotFound(c, id)
		case errors.Is(err, store.ErrAmbiguousID):
			respondSessionAmbiguousID(c, id)
		default:
			respondInternalErr(c, h.logger, err)
		}
		return
	}
	c.JSON(http.StatusOK, toSessionDetail(sess))
}

func (h *handlers) deleteSession(c *gin.Context) {
	id := c.Param("id")

	// id may be an unambiguous prefix; resolve it to the full ID before
	// publishing session.deleted, so WS clients matching on payload.id
	// always see the canonical ID (mirrors deleteMessage).
	sess, err := h.store.GetSession(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			respondSessionNotFound(c, id)
		case errors.Is(err, store.ErrAmbiguousID):
			respondSessionAmbiguousID(c, id)
		default:
			respondInternalErr(c, h.logger, err)
		}
		return
	}

	if err := h.store.DeleteSession(c.Request.Context(), id); err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			respondSessionNotFound(c, id)
		case errors.Is(err, store.ErrAmbiguousID):
			respondSessionAmbiguousID(c, id)
		default:
			respondInternalErr(c, h.logger, err)
		}
		return
	}
	h.bus.Publish(events.SessionDeleted(sess.ID))
	c.Status(http.StatusNoContent)
}

func (h *handlers) clearSessions(c *gin.Context) {
	if c.Query("confirm") != "true" {
		respondError(c, http.StatusBadRequest, "confirmation_required", "bulk delete requires ?confirm=true")
		return
	}
	if err := h.store.ClearSessions(c.Request.Context()); err != nil {
		respondInternalErr(c, h.logger, err)
		return
	}
	h.bus.Publish(events.SessionsCleared())
	c.Status(http.StatusNoContent)
}
