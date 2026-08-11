package store

import "time"

// TranscriptLine is one line of an SMTP session's raw protocol transcript
// (SPEC.md §4/§6, M8.4). Direction is 'C' (client -> server) or 'S'
// (server -> client). AUTH argument lines are redacted before being stored
// here (see internal/smtp/session.go) — this type never carries raw
// credential bytes.
type TranscriptLine struct {
	Direction byte
	Line      string
	Position  int
	Ts        time.Time
}

// Session is a full record of one accepted SMTP connection, independent of
// whether it ever produced a stored message (M8.4). EndedAt/MessageID are
// nil until the connection closes / a message is successfully stored.
type Session struct {
	ID         string
	ClientIP   string
	ClientHELO string
	StartedAt  time.Time
	EndedAt    *time.Time
	// Status is one of "completed", "rejected", "aborted", "timeout" once
	// EndedAt is set; empty while the session is still in progress.
	Status     string
	MessageID  *string
	Transcript []TranscriptLine
}

// SessionSummary is the Session shape without its transcript, for list
// responses and the session.started/session.completed WS event payloads
// (SPEC.md §5.5).
type SessionSummary struct {
	ID         string
	ClientIP   string
	ClientHELO string
	StartedAt  time.Time
	EndedAt    *time.Time
	Status     string
	MessageID  *string
}

// Sort orders for SessionListFilter.Sort. SortStartedAtDesc is the default.
const (
	SortStartedAtDesc = "started_at_desc"
	SortStartedAtAsc  = "started_at_asc"
)

// SessionListFilter controls pagination and filtering for ListSessions, per
// SPEC.md §5.2's GET /api/v1/sessions query params.
type SessionListFilter struct {
	Limit  int
	Offset int

	Status   string
	ClientIP string

	// Since, Until bound StartedAt (inclusive); zero value means unset.
	Since, Until time.Time

	// Sort is one of the Sort* constants; "" defaults to SortStartedAtDesc.
	Sort string
}

// NewSessionSummary builds the summary shape for sess.
func NewSessionSummary(sess *Session) SessionSummary {
	return SessionSummary{
		ID:         sess.ID,
		ClientIP:   sess.ClientIP,
		ClientHELO: sess.ClientHELO,
		StartedAt:  sess.StartedAt,
		EndedAt:    sess.EndedAt,
		Status:     sess.Status,
		MessageID:  sess.MessageID,
	}
}
