package store

import (
	"regexp"
	"strings"
	"time"
)

// MessageSummary is the Message (summary) shape from SPEC.md §5.1. It also
// backs the JSON payload of a message.created WebSocket event (SPEC.md
// §5.5), which is why it lives in internal/store rather than internal/api:
// both internal/api and internal/smtp need to build this exact shape, and
// neither package may import the other.
type MessageSummary struct {
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
	Read            bool     `json:"read"`
	Tags            []string `json:"tags"`
	Preview         string   `json:"preview"`
}

const summaryPreviewMaxLen = 120

// addrStrings flattens a slice of Address to their bare address strings.
func addrStrings(addrs []Address) []string {
	out := make([]string, len(addrs))
	for i, a := range addrs {
		out[i] = a.Address
	}
	return out
}

// tagsOrEmpty avoids sending `"tags": null` to clients (nil vs. empty slice
// both mean "no tags", but JSON null forces every caller to null-check).
func tagsOrEmpty(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}

var summaryHTMLTagRe = regexp.MustCompile(`(?s)<[^>]*>`)
var summaryWhitespaceRe = regexp.MustCompile(`\s+`)

func stripHTMLTags(html string) string {
	return summaryHTMLTagRe.ReplaceAllString(html, " ")
}

func collapseWhitespace(s string) string {
	return strings.TrimSpace(summaryWhitespaceRe.ReplaceAllString(s, " "))
}

// messagePreview computes a short, plain-text-only truncated preview: the
// first ~120 chars of TextBody, or of HTMLBody with tags stripped if no
// TextBody exists.
func messagePreview(msg *Message) string {
	text := strings.TrimSpace(msg.TextBody)
	if text == "" {
		text = strings.TrimSpace(stripHTMLTags(msg.HTMLBody))
	}
	text = collapseWhitespace(text)
	r := []rune(text)
	if len(r) > summaryPreviewMaxLen {
		return string(r[:summaryPreviewMaxLen])
	}
	return string(r)
}

// NewMessageSummary builds the summary JSON shape for msg.
func NewMessageSummary(msg *Message) MessageSummary {
	from := ""
	if len(msg.From) > 0 {
		from = msg.From[0].Address
	}
	return MessageSummary{
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
		Read:            msg.Read,
		Tags:            tagsOrEmpty(msg.Tags),
		Preview:         messagePreview(msg),
	}
}
