package store

import (
	"context"
	"errors"
)

// ErrNotFound is returned by Get/Delete when no message exists with the
// given ID.
var ErrNotFound = errors.New("store: message not found")

// ListFilter controls pagination for List. It is intentionally minimal for
// M1.0 — later milestones (M2.0/M3.0) extend it with query filters
// (from/to/subject/full-text) without breaking this signature, since new
// fields default to their zero value (no filtering).
type ListFilter struct {
	Limit  int
	Offset int
}

// MessageStore is the storage-agnostic interface every backend (in-memory
// here, SQLite in M2.0) implements. No SQL-specific types leak through this
// interface, so callers (the SMTP server, and later the REST API) never
// depend on a concrete backend.
type MessageStore interface {
	Save(ctx context.Context, msg *Message) error
	Get(ctx context.Context, id string) (*Message, error)
	// List returns the page of messages selected by filter (newest first)
	// and the total number of messages matching (ignoring pagination).
	List(ctx context.Context, filter ListFilter) ([]*Message, int, error)
	Delete(ctx context.Context, id string) error
	Clear(ctx context.Context) error
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
