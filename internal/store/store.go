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

	// Query performs an FTS5 match against subject/from/to/text_body
	// (messages_fts).
	Query string
	// From, To, Subject are case-insensitive substring filters.
	From, To, Subject string
	// Since, Until bound received_at (inclusive); zero value means unset.
	Since, Until time.Time
	// Sort is one of the Sort* constants; "" defaults to SortReceivedAtDesc.
	Sort string
}

// Stats summarizes the store's current contents for GET /api/v1/stats.
type Stats struct {
	TotalMessages    int
	TotalSizeBytes   int64
	OldestReceivedAt *time.Time
	NewestReceivedAt *time.Time
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
	// Stats returns a snapshot summary of the store's current contents.
	Stats(ctx context.Context) (Stats, error)
	// Ping verifies the underlying storage is reachable, for health checks.
	Ping(ctx context.Context) error
}

// Publisher is notified after a message is durably saved. It stands in for
// the full in-process event bus that M7.0 builds (message.created/deleted/
// cleared over a pub/sub bus) — for M1.0, /internal/smtp depends on nothing
// more than this single method, satisfying SPEC.md §2.3 point 5's "depends
// only on the MessageStore interface and the event bus" without building
// pub/sub prematurely.
type Publisher interface {
	Publish(ctx context.Context, msg *Message)
}

// NoopPublisher discards every event. It is the default Publisher until
// M7.0 wires in a real event bus.
type NoopPublisher struct{}

func (NoopPublisher) Publish(context.Context, *Message) {}
