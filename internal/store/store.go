package store

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned by Get/Delete when no message exists with the
// given ID (or ID prefix, see IDLength).
var ErrNotFound = errors.New("store: message not found")

// ErrAmbiguousID is returned by Get/Delete when a short ID prefix matches
// more than one stored message.
var ErrAmbiguousID = errors.New("store: ambiguous message id prefix")

// ErrInvalidQuery is returned by List when ListFilter.Query is not valid
// FTS5 query syntax (SPEC.md §5.2's `q` param is passed verbatim into
// SQLite's FTS5 MATCH operator, so a malformed boolean/phrase/column-filter
// expression surfaces here rather than as a generic storage failure).
// Callers should map this to a 400 response with a user-facing message,
// never the raw driver error text, which is an internal implementation
// detail (SQL dialect, column names) that would leak through otherwise.
var ErrInvalidQuery = errors.New("store: invalid search query")

// ErrInvalidTag is returned by AddTag/RemoveTag when the given tag is empty
// or whitespace-only after trimming.
var ErrInvalidTag = errors.New("store: invalid tag")

// IDLength is the length, in hex characters, of a full message ID produced
// by NewID. Get and Delete treat any shorter string as a prefix to resolve
// against stored IDs (docker-CLI-style short references), and any string of
// this length or longer as a full ID looked up directly.
const IDLength = 24

// Sort orders for ListFilter.Sort. ReceivedAtDesc is the default.
const (
	SortReceivedAtDesc = "received_at_desc"
	SortReceivedAtAsc  = "received_at_asc"
)

// ListFilter controls pagination and filtering for List, per SPEC.md §5.2's
// GET /api/v1/messages query params. Zero-value fields mean "no filtering" so
// M1.0 callers (which only ever set Limit/Offset) keep working unchanged.
type ListFilter struct {
	Limit  int
	Offset int

	// Query performs an FTS5 match against subject/from/to/cc/bcc/text_body
	// (messages_fts).
	Query string
	// From, To, Subject, Cc, Bcc are case-insensitive substring filters.
	From, To, Subject, Cc, Bcc string
	// Since, Until bound received_at (inclusive); zero value means unset.
	Since, Until time.Time
	// Sort is one of the Sort* constants; "" defaults to SortReceivedAtDesc.
	Sort string
	// Tag, when non-empty, matches messages carrying this exact tag
	// (case-sensitive exact match against one entry of Message.Tags). Kept
	// as sugar for Tags: []string{Tag} — set either, not both.
	Tag string
	// Tags, when non-empty, matches messages against multiple tags at once,
	// combined per TagMode. If both Tag and Tags are set, Tags wins.
	Tags []string
	// TagMode is "any" (OR, the default/zero value) or "all" (AND),
	// controlling how Tags are combined. Ignored when len(Tags) < 2.
	TagMode string
	// Read, when non-nil, filters to messages with Read == *Read.
	Read *bool
	// HasAttachments, when non-nil, filters to messages whose
	// AttachmentCount is >0 (true) or 0 (false).
	HasAttachments *bool
	// ParseWarning, when non-nil, filters to messages with
	// ParseWarning == *ParseWarning.
	ParseWarning *bool
}

// Stats summarizes the store's current contents for GET /api/v1/stats.
type Stats struct {
	TotalMessages    int
	TotalSizeBytes   int64
	OldestReceivedAt *time.Time
	NewestReceivedAt *time.Time

	// UnreadCount, AttachmentCount, ParseWarningCount are cheap aggregate
	// counts backing the Web UI sidebar's mailbox filter section (M6.1) —
	// computed without loading full message lists.
	UnreadCount       int
	AttachmentCount   int
	ParseWarningCount int
}

// TagCount is one row of GET /api/v1/tags: a distinct tag value currently in
// use and how many messages carry it.
type TagCount struct {
	Tag   string
	Count int
}

// MessageStore is the storage-agnostic interface every backend (in-memory
// here, SQLite in M2.0) implements. No SQL-specific types leak through this
// interface, so callers (the SMTP server, and later the REST API) never
// depend on a concrete backend.
type MessageStore interface {
	Save(ctx context.Context, msg *Message) error
	// Get accepts either a full message ID or an unambiguous prefix of one
	// (any string shorter than IDLength), docker-CLI-style. A prefix
	// matching zero messages returns ErrNotFound; a prefix matching more
	// than one returns ErrAmbiguousID.
	Get(ctx context.Context, id string) (*Message, error)
	// List returns the page of messages selected by filter (newest first)
	// and the total number of messages matching (ignoring pagination).
	List(ctx context.Context, filter ListFilter) ([]*Message, int, error)
	// Delete accepts a full ID or unambiguous prefix, per Get.
	Delete(ctx context.Context, id string) error
	Clear(ctx context.Context) error
	// MarkRead sets the read flag (full ID or unambiguous prefix, per Get)
	// to the given value. Idempotent — setting a message's read flag to a
	// value it already holds is a no-op success.
	MarkRead(ctx context.Context, id string, read bool) error
	// AddTag adds tag (trimmed) to the message's tag set (full ID or
	// unambiguous prefix, per Get). Adding a tag the message already has is
	// a no-op success. Returns ErrInvalidTag if tag is empty/whitespace-only
	// after trimming.
	AddTag(ctx context.Context, id, tag string) error
	// RemoveTag removes tag (trimmed) from the message's tag set (full ID or
	// unambiguous prefix, per Get). Removing a tag the message doesn't have
	// is a no-op success. Returns ErrInvalidTag if tag is empty/whitespace-only
	// after trimming.
	RemoveTag(ctx context.Context, id, tag string) error
	// Stats returns a snapshot summary of the store's current contents.
	Stats(ctx context.Context) (Stats, error)
	// Tags returns every distinct tag currently in use with its message
	// count, for GET /api/v1/tags.
	Tags(ctx context.Context) ([]TagCount, error)
	// Ping verifies the underlying storage is reachable, for health checks.
	Ping(ctx context.Context) error
}
