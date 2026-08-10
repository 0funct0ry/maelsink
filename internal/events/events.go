// Package events implements maelsink's in-process event bus (SPEC.md §2.2,
// §5.5): a simple pub/sub broadcaster over Go channels, with no external
// broker, that lets mutating code paths (SMTP ingestion, the REST API, the
// retention sweeper) notify the WebSocket hub of message.created,
// message.deleted, and messages.cleared events.
package events

// Type identifies the kind of event, matching the JSON "type" field of the
// WebSocket wire frames in SPEC.md §5.5.
type Type string

const (
	TypeMessageCreated     Type = "message.created"
	TypeMessageDeleted     Type = "message.deleted"
	TypeMessagesCleared    Type = "messages.cleared"
	TypeMessageTagsUpdated Type = "message.tags_updated"
)

// Event is the bus-internal envelope. Payload is whatever the caller wants
// forwarded as the JSON frame's "payload" field. Kept as any so this
// package has no dependency on internal/store's message shape — callers
// already know how to build the right payload for their event type.
type Event struct {
	Type    Type
	Payload any
}

// deletedPayload is the {"id": "msg_..."} shape SPEC.md §5.5 defines for
// message.deleted.
type deletedPayload struct {
	ID string `json:"id"`
}

// tagsUpdatedPayload is the {"id": "msg_...", "tags": [...]} shape SPEC.md
// §5.5 defines for message.tags_updated.
type tagsUpdatedPayload struct {
	ID   string   `json:"id"`
	Tags []string `json:"tags"`
}

// MessageCreated builds a message.created event carrying payload (a message
// summary) as its JSON payload.
func MessageCreated(payload any) Event {
	return Event{Type: TypeMessageCreated, Payload: payload}
}

// MessageDeleted builds a message.deleted event for the given (full)
// message ID.
func MessageDeleted(id string) Event {
	return Event{Type: TypeMessageDeleted, Payload: deletedPayload{ID: id}}
}

// MessagesCleared builds a messages.cleared event with an empty payload.
func MessagesCleared() Event {
	return Event{Type: TypeMessagesCleared, Payload: struct{}{}}
}

// MessageTagsUpdated builds a message.tags_updated event for the given
// (full) message ID and its current tag set. tags is copied into the
// payload as-is; callers should pass []string{} rather than nil for "no
// tags" to match the no-null convention used elsewhere (e.g.
// store.MessageSummary.Tags).
func MessageTagsUpdated(id string, tags []string) Event {
	return Event{Type: TypeMessageTagsUpdated, Payload: tagsUpdatedPayload{ID: id, Tags: tags}}
}
