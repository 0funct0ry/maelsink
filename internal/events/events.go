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
	TypeTagRenamed         Type = "tag.renamed"
	TypeTagRecolored       Type = "tag.recolored"
	TypeTagCreated         Type = "tag.created"
	TypeTagDeleted         Type = "tag.deleted"
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

// tagRenamedPayload is the {"name": "...", "new_name": "...", "merged": bool}
// shape for tag.renamed.
type tagRenamedPayload struct {
	Name    string `json:"name"`
	NewName string `json:"new_name"`
	Merged  bool   `json:"merged"`
}

// tagRecoloredPayload is the {"name": "...", "color": "..."} shape for
// tag.recolored.
type tagRecoloredPayload struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// tagCreatedPayload is the {"name": "...", "color": "..."} shape for
// tag.created.
type tagCreatedPayload struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// tagDeletedPayload is the {"name": "..."} shape for tag.deleted.
type tagDeletedPayload struct {
	Name string `json:"name"`
}

// TagRenamed builds a tag.renamed event. merged is true when newName
// already existed and the two tags were merged rather than oldName simply
// being renamed.
func TagRenamed(oldName, newName string, merged bool) Event {
	return Event{Type: TypeTagRenamed, Payload: tagRenamedPayload{Name: oldName, NewName: newName, Merged: merged}}
}

// TagRecolored builds a tag.recolored event.
func TagRecolored(name, color string) Event {
	return Event{Type: TypeTagRecolored, Payload: tagRecoloredPayload{Name: name, Color: color}}
}

// TagCreated builds a tag.created event.
func TagCreated(name, color string) Event {
	return Event{Type: TypeTagCreated, Payload: tagCreatedPayload{Name: name, Color: color}}
}

// TagDeleted builds a tag.deleted event, used for both the untag-only and
// delete-with-messages variants (the payload shape doesn't need to
// distinguish them; DeleteTagWithMessages also publishes a MessageDeleted
// per removed message).
func TagDeleted(name string) Event {
	return Event{Type: TypeTagDeleted, Payload: tagDeletedPayload{Name: name}}
}
