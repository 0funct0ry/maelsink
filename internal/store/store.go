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

// ErrTagNotFound is returned by RenameTag/RecolorTag/DeleteTag/
// DeleteTagWithMessages when no tag exists with the given name.
var ErrTagNotFound = errors.New("store: tag not found")

// ErrTagExists is returned by CreateTag when a tag with the given name
// already exists.
var ErrTagExists = errors.New("store: tag already exists")

// TagColors is the fixed set of persisted color tokens a tag may have.
// CreateTag/RecolorTag reject any other value with ErrInvalidTag.
var TagColors = []string{"indigo", "emerald", "amber", "rose", "cyan", "fuchsia", "lime", "orange"}

// IsValidTagColor reports whether color is one of TagColors.
func IsValidTagColor(color string) bool {
	for _, c := range TagColors {
		if c == color {
			return true
		}
	}
	return false
}

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

// TagStats is one row of GET /api/v1/tags: a persisted tag's name and color,
// plus how many messages currently carry it and when it was last used. Count
// is 0 and LastUsed is nil for a tag with no messages (a tag created via
// CreateTag with none attached yet, or one every message has been untagged
// from).
type TagStats struct {
	Name     string
	Color    string
	Count    int
	LastUsed *time.Time
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
	// ListTagsWithStats returns every persisted tag (including ones with no
	// messages attached) with its color, message count, and last-used time,
	// for GET /api/v1/tags. Ordered by Count descending, then LastUsed
	// descending, then Name ascending.
	ListTagsWithStats(ctx context.Context) ([]TagStats, error)
	// RenameTag renames oldName (trimmed) to newName (trimmed) across every
	// message carrying it and the tags table itself. Returns ErrInvalidTag
	// if either is empty/whitespace-only, ErrTagNotFound if oldName doesn't
	// exist. If newName already exists, the two tags are merged instead of
	// erroring: every message carrying oldName gains newName (deduped),
	// oldName's tags row is deleted, and newName's existing row (color,
	// created_at) is left untouched.
	RenameTag(ctx context.Context, oldName, newName string) error
	// RecolorTag updates name's persisted color. Returns ErrInvalidTag if
	// name is empty/whitespace-only or color is not one of TagColors, and
	// ErrTagNotFound if name doesn't exist.
	RecolorTag(ctx context.Context, name, color string) error
	// CreateTag inserts a new tag with no messages attached yet. Returns
	// ErrInvalidTag if name is empty/whitespace-only or color is not one of
	// TagColors, and ErrTagExists if name already exists.
	CreateTag(ctx context.Context, name, color string) error
	// DeleteTag removes name from every message's tag set (untags, does not
	// delete messages) and deletes its tags row. Returns ErrInvalidTag if
	// name is empty/whitespace-only, ErrTagNotFound if name doesn't exist.
	DeleteTag(ctx context.Context, name string) error
	// DeleteTagWithMessages deletes every message carrying name (via the
	// same path as Delete, so attachments are cleaned up identically) and
	// then deletes its tags row. Returns ErrInvalidTag if name is
	// empty/whitespace-only, ErrTagNotFound if name doesn't exist.
	DeleteTagWithMessages(ctx context.Context, name string) error
	// Ping verifies the underlying storage is reachable, for health checks.
	Ping(ctx context.Context) error
}
