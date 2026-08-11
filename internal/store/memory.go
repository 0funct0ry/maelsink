package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryStore is an in-memory MessageStore. It is the only backend for
// M1.0; M2.0 replaces it with a SQLite-backed implementation of the same
// MessageStore interface.
type MemoryStore struct {
	mu          sync.RWMutex
	messages    []*Message
	byID        map[string]int // id -> index into messages
	tagRegistry map[string]tagMeta

	sessions []*Session
	sessByID map[string]int // id -> index into sessions
}

// tagMeta holds a persisted tag's color/created_at, independent of whether
// any message currently carries it (a tag created via CreateTag with no
// messages attached still needs somewhere to live).
type tagMeta struct {
	color     string
	createdAt time.Time
}

// NewMemoryStore returns an empty, ready-to-use MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		byID:        make(map[string]int),
		tagRegistry: make(map[string]tagMeta),
		sessByID:    make(map[string]int),
	}
}

// Save stores msg, generating an ID if msg.ID is empty. Save takes ownership
// of the pointer; callers must not mutate msg after calling Save.
func (s *MemoryStore) Save(_ context.Context, msg *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if msg.ID == "" {
		msg.ID = NewID()
	}

	for _, t := range msg.Tags {
		s.ensureTagRegistered(t)
	}

	if idx, ok := s.byID[msg.ID]; ok {
		s.messages[idx] = msg
		return nil
	}

	s.byID[msg.ID] = len(s.messages)
	s.messages = append(s.messages, msg)
	return nil
}

// Get returns the message with the given ID or unambiguous ID prefix (see
// IDLength), or ErrNotFound/ErrAmbiguousID.
func (s *MemoryStore) Get(_ context.Context, id string) (*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	full, err := resolveID(s.byID, id)
	if err != nil {
		return nil, err
	}
	copy := *s.messages[s.byID[full]]
	copy.AttachmentCount = len(copy.Attachments) + len(copy.InlineImages)
	return &copy, nil
}

// resolveID looks up id in byID (keyed by full message ID). Strings of
// IDLength or longer are matched exactly (the common case, and the fast
// path — no prefix scan). Shorter strings are resolved as a prefix: zero
// matches is ErrNotFound, more than one is ErrAmbiguousID.
func resolveID(byID map[string]int, id string) (string, error) {
	if len(id) >= IDLength {
		if _, ok := byID[id]; ok {
			return id, nil
		}
		return "", ErrNotFound
	}

	var match string
	count := 0
	for full := range byID {
		if strings.HasPrefix(full, id) {
			match = full
			count++
			if count > 1 {
				return "", ErrAmbiguousID
			}
		}
	}
	if count == 0 {
		return "", ErrNotFound
	}
	return match, nil
}

// List returns messages matching filter (newest-first by default, or
// oldest-first when filter.Sort == SortReceivedAtAsc), paginated, plus the
// total count of matches ignoring pagination.
func (s *MemoryStore) List(_ context.Context, filter ListFilter) ([]*Message, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matched []*Message
	for _, msg := range s.messages {
		if messageMatchesFilter(msg, filter) {
			// Return a shallow copy so callers mutating/reading
			// AttachmentCount never race with concurrent List/Get calls
			// sharing the same underlying *Message.
			copy := *msg
			copy.AttachmentCount = len(msg.Attachments) + len(msg.InlineImages)
			matched = append(matched, &copy)
		}
	}

	// Build newest-first order (default) without mutating s.messages.
	ordered := make([]*Message, len(matched))
	for i, msg := range matched {
		ordered[len(matched)-1-i] = msg
	}
	if filter.Sort == SortReceivedAtAsc {
		ordered = matched
	}

	total := len(ordered)

	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= total {
		return []*Message{}, total, nil
	}
	ordered = ordered[offset:]

	if filter.Limit > 0 && filter.Limit < len(ordered) {
		ordered = ordered[:filter.Limit]
	}
	return ordered, total, nil
}

func messageMatchesFilter(msg *Message, filter ListFilter) bool {
	if filter.From != "" && !addrsContain(msg.From, filter.From) {
		return false
	}
	if filter.To != "" && !addrsContain(msg.To, filter.To) {
		return false
	}
	if filter.Subject != "" && !strings.Contains(strings.ToLower(msg.Subject), strings.ToLower(filter.Subject)) {
		return false
	}
	if filter.Cc != "" && !addrsContain(msg.Cc, filter.Cc) {
		return false
	}
	if filter.Bcc != "" && !addrsContain(msg.Bcc, filter.Bcc) {
		return false
	}
	if filter.Query != "" {
		q := strings.ToLower(filter.Query)
		if !strings.Contains(strings.ToLower(msg.Subject), q) &&
			!strings.Contains(strings.ToLower(msg.TextBody), q) &&
			!addrsContain(msg.From, filter.Query) &&
			!addrsContain(msg.To, filter.Query) {
			return false
		}
	}
	if !filter.Since.IsZero() && msg.ReceivedAt.Before(filter.Since) {
		return false
	}
	if !filter.Until.IsZero() && msg.ReceivedAt.After(filter.Until) {
		return false
	}
	if tags := effectiveTags(filter); len(tags) > 0 {
		if filter.TagMode == "all" {
			for _, t := range tags {
				if !tagsContain(msg.Tags, t) {
					return false
				}
			}
		} else {
			matched := false
			for _, t := range tags {
				if tagsContain(msg.Tags, t) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
	}
	if filter.Read != nil && msg.Read != *filter.Read {
		return false
	}
	if filter.HasAttachments != nil {
		has := len(msg.Attachments)+len(msg.InlineImages) > 0
		if has != *filter.HasAttachments {
			return false
		}
	}
	if filter.ParseWarning != nil && msg.ParseWarning != *filter.ParseWarning {
		return false
	}
	return true
}

// effectiveTags returns filter.Tags, falling back to filter.Tag as sugar for
// a single-element slice when Tags is unset.
func effectiveTags(filter ListFilter) []string {
	if len(filter.Tags) > 0 {
		return filter.Tags
	}
	if filter.Tag != "" {
		return []string{filter.Tag}
	}
	return nil
}

func tagsContain(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

func addrsContain(addrs []Address, substr string) bool {
	substr = strings.ToLower(substr)
	for _, a := range addrs {
		if strings.Contains(strings.ToLower(a.Address), substr) {
			return true
		}
	}
	return false
}

// Stats returns a snapshot summary of the store's current contents.
func (s *MemoryStore) Stats(_ context.Context) (Stats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := Stats{TotalMessages: len(s.messages)}
	for _, msg := range s.messages {
		stats.TotalSizeBytes += msg.Size
		rt := msg.ReceivedAt
		if stats.OldestReceivedAt == nil || rt.Before(*stats.OldestReceivedAt) {
			stats.OldestReceivedAt = &rt
		}
		if stats.NewestReceivedAt == nil || rt.After(*stats.NewestReceivedAt) {
			stats.NewestReceivedAt = &rt
		}
		if !msg.Read {
			stats.UnreadCount++
		}
		if len(msg.Attachments)+len(msg.InlineImages) > 0 {
			stats.AttachmentCount++
		}
		if msg.ParseWarning {
			stats.ParseWarningCount++
		}
	}
	return stats, nil
}

// hashDefaultColor deterministically maps a tag name to one of TagColors,
// bit-for-bit identical to web/src/lib/tagColor.ts's hash — used to give a
// tag registered implicitly (via AddTag, never CreateTag) a stable color,
// mirroring the SQLite backend's migration backfill.
func hashDefaultColor(name string) string {
	h := uint32(2166136261)
	for i := 0; i < len(name); i++ {
		h ^= uint32(name[i])
		h *= 16777619
	}
	return TagColors[int(h)%len(TagColors)]
}

// ensureTagRegistered gives tag a registry entry (with a hash-derived
// default color) if it doesn't already have one. Callers must hold s.mu for
// writing.
func (s *MemoryStore) ensureTagRegistered(tag string) {
	if _, ok := s.tagRegistry[tag]; !ok {
		s.tagRegistry[tag] = tagMeta{color: hashDefaultColor(tag), createdAt: time.Now()}
	}
}

// ListTagsWithStats returns every persisted tag with its usage stats. See
// MessageStore.ListTagsWithStats.
func (s *MemoryStore) ListTagsWithStats(_ context.Context) ([]TagStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := make(map[string]int)
	lastUsed := make(map[string]time.Time)
	for _, msg := range s.messages {
		for _, t := range msg.Tags {
			counts[t]++
			if lu, ok := lastUsed[t]; !ok || msg.ReceivedAt.After(lu) {
				lastUsed[t] = msg.ReceivedAt
			}
		}
	}

	result := make([]TagStats, 0, len(s.tagRegistry))
	for name, meta := range s.tagRegistry {
		ts := TagStats{Name: name, Color: meta.color, Count: counts[name]}
		if lu, ok := lastUsed[name]; ok {
			luCopy := lu
			ts.LastUsed = &luCopy
		}
		result = append(result, ts)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		li, lj := result[i].LastUsed, result[j].LastUsed
		switch {
		case li == nil && lj == nil:
		case li == nil:
			return false
		case lj == nil:
			return true
		case !li.Equal(*lj):
			return li.After(*lj)
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// RenameTag renames oldName to newName, merging into newName if it already
// exists. See MessageStore.RenameTag.
func (s *MemoryStore) RenameTag(_ context.Context, oldName, newName string) error {
	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if oldName == "" || newName == "" {
		return ErrInvalidTag
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	oldMeta, ok := s.tagRegistry[oldName]
	if !ok {
		return ErrTagNotFound
	}
	_, merge := s.tagRegistry[newName]

	for _, msg := range s.messages {
		if !tagsContain(msg.Tags, oldName) {
			continue
		}
		next := make([]string, 0, len(msg.Tags))
		hasNew := false
		for _, t := range msg.Tags {
			if t == oldName {
				continue
			}
			if t == newName {
				hasNew = true
			}
			next = append(next, t)
		}
		if !hasNew {
			next = append(next, newName)
		}
		msg.Tags = next
	}

	delete(s.tagRegistry, oldName)
	if !merge {
		s.tagRegistry[newName] = oldMeta
	}
	return nil
}

// RecolorTag updates name's persisted color. See MessageStore.RecolorTag.
func (s *MemoryStore) RecolorTag(_ context.Context, name, color string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidTag
	}
	if !IsValidTagColor(color) {
		return ErrInvalidTag
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	meta, ok := s.tagRegistry[name]
	if !ok {
		return ErrTagNotFound
	}
	meta.color = color
	s.tagRegistry[name] = meta
	return nil
}

// CreateTag inserts a new tag with no messages attached. See
// MessageStore.CreateTag.
func (s *MemoryStore) CreateTag(_ context.Context, name, color string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidTag
	}
	if !IsValidTagColor(color) {
		return ErrInvalidTag
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tagRegistry[name]; ok {
		return ErrTagExists
	}
	s.tagRegistry[name] = tagMeta{color: color, createdAt: time.Now()}
	return nil
}

// DeleteTag removes name from every message's tag set and deletes its
// registry entry. See MessageStore.DeleteTag.
func (s *MemoryStore) DeleteTag(_ context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidTag
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tagRegistry[name]; !ok {
		return ErrTagNotFound
	}
	for _, msg := range s.messages {
		if !tagsContain(msg.Tags, name) {
			continue
		}
		next := make([]string, 0, len(msg.Tags))
		for _, t := range msg.Tags {
			if t != name {
				next = append(next, t)
			}
		}
		msg.Tags = next
	}
	delete(s.tagRegistry, name)
	return nil
}

// DeleteTagWithMessages deletes every message carrying name, then its
// registry entry. See MessageStore.DeleteTagWithMessages.
func (s *MemoryStore) DeleteTagWithMessages(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidTag
	}

	s.mu.Lock()
	if _, ok := s.tagRegistry[name]; !ok {
		s.mu.Unlock()
		return ErrTagNotFound
	}
	var ids []string
	for _, msg := range s.messages {
		if tagsContain(msg.Tags, name) {
			ids = append(ids, msg.ID)
		}
	}
	s.mu.Unlock()

	for _, id := range ids {
		if err := s.Delete(ctx, id); err != nil && err != ErrNotFound {
			return err
		}
	}

	s.mu.Lock()
	delete(s.tagRegistry, name)
	s.mu.Unlock()
	return nil
}

// Ping always succeeds for the in-memory store.
func (s *MemoryStore) Ping(_ context.Context) error {
	return nil
}

// Delete removes the message with the given ID or unambiguous ID prefix, or
// returns ErrNotFound/ErrAmbiguousID.
func (s *MemoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	full, err := resolveID(s.byID, id)
	if err != nil {
		return err
	}
	idx := s.byID[full]

	s.messages = append(s.messages[:idx], s.messages[idx+1:]...)
	delete(s.byID, full)
	for i := idx; i < len(s.messages); i++ {
		s.byID[s.messages[i].ID] = i
	}
	s.clearSessionMessageID(full)
	return nil
}

// clearSessionMessageID nils out MessageID on any session pointing at
// messageID (M8.4), so a deleted message never leaves a dangling
// cross-link — the session record + transcript are independently
// retained.
func (s *MemoryStore) clearSessionMessageID(messageID string) {
	for _, sess := range s.sessions {
		if sess.MessageID != nil && *sess.MessageID == messageID {
			sess.MessageID = nil
		}
	}
}

// MarkRead sets the read flag of the message with the given ID or
// unambiguous ID prefix, or returns ErrNotFound/ErrAmbiguousID.
func (s *MemoryStore) MarkRead(_ context.Context, id string, read bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	full, err := resolveID(s.byID, id)
	if err != nil {
		return err
	}
	s.messages[s.byID[full]].Read = read
	return nil
}

// AddTag adds tag to the message's tag set. See MessageStore.AddTag.
func (s *MemoryStore) AddTag(_ context.Context, id, tag string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return ErrInvalidTag
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	full, err := resolveID(s.byID, id)
	if err != nil {
		return err
	}
	msg := s.messages[s.byID[full]]
	if tagsContain(msg.Tags, tag) {
		return nil
	}
	// Build a fresh slice rather than appending in place: List/Get hand out
	// shallow copies of *Message that share this slice's backing array, so
	// an in-place append could race with a concurrent reader under -race.
	next := make([]string, len(msg.Tags), len(msg.Tags)+1)
	copy(next, msg.Tags)
	next = append(next, tag)
	msg.Tags = next
	s.ensureTagRegistered(tag)
	return nil
}

// RemoveTag removes tag from the message's tag set. See MessageStore.RemoveTag.
func (s *MemoryStore) RemoveTag(_ context.Context, id, tag string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return ErrInvalidTag
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	full, err := resolveID(s.byID, id)
	if err != nil {
		return err
	}
	msg := s.messages[s.byID[full]]
	if !tagsContain(msg.Tags, tag) {
		return nil
	}
	next := make([]string, 0, len(msg.Tags))
	for _, t := range msg.Tags {
		if t != tag {
			next = append(next, t)
		}
	}
	msg.Tags = next
	return nil
}

// Clear removes every stored message.
func (s *MemoryStore) Clear(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = nil
	s.byID = make(map[string]int)
	for _, sess := range s.sessions {
		sess.MessageID = nil
	}
	return nil
}

// NewID generates a random, URL-safe ID, used for messages, attachments, and
// inline images alike.
func NewID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand.Read failing is effectively impossible on supported
		// platforms; panicking here surfaces it loudly rather than silently
		// handing out colliding/empty IDs.
		panic("store: failed to generate message id: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// CreateSession persists a finished session. See MessageStore.CreateSession.
func (s *MemoryStore) CreateSession(_ context.Context, sess *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sess.ID == "" {
		sess.ID = NewID()
	}

	copy := *sess
	if idx, ok := s.sessByID[sess.ID]; ok {
		s.sessions[idx] = &copy
		return nil
	}
	s.sessByID[sess.ID] = len(s.sessions)
	s.sessions = append(s.sessions, &copy)
	return nil
}

// AppendSessionLine persists a single transcript line for an
// already-created session. See MessageStore.AppendSessionLine.
func (s *MemoryStore) AppendSessionLine(_ context.Context, sessionID string, line TranscriptLine) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx, ok := s.sessByID[sessionID]
	if !ok {
		return ErrNotFound
	}
	s.sessions[idx].Transcript = append(s.sessions[idx].Transcript, line)
	return nil
}

// GetSession returns the session with the given ID or unambiguous ID
// prefix, or ErrNotFound/ErrAmbiguousID.
func (s *MemoryStore) GetSession(_ context.Context, id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	full, err := resolveID(s.sessByID, id)
	if err != nil {
		return nil, err
	}
	copy := *s.sessions[s.sessByID[full]]
	return &copy, nil
}

// ListSessions returns sessions matching filter (newest-started first by
// default), paginated, plus the total count of matches ignoring pagination.
func (s *MemoryStore) ListSessions(_ context.Context, filter SessionListFilter) ([]*SessionSummary, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matched []*Session
	for _, sess := range s.sessions {
		if sessionMatchesFilter(sess, filter) {
			matched = append(matched, sess)
		}
	}

	ordered := make([]*Session, len(matched))
	for i, sess := range matched {
		ordered[len(matched)-1-i] = sess
	}
	if filter.Sort == SortStartedAtAsc {
		ordered = matched
	}

	total := len(ordered)

	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= total {
		return []*SessionSummary{}, total, nil
	}
	ordered = ordered[offset:]

	if filter.Limit > 0 && filter.Limit < len(ordered) {
		ordered = ordered[:filter.Limit]
	}

	out := make([]*SessionSummary, len(ordered))
	for i, sess := range ordered {
		summary := NewSessionSummary(sess)
		out[i] = &summary
	}
	return out, total, nil
}

// DeleteSession removes the session with the given full ID or unambiguous
// ID prefix, or ErrNotFound/ErrAmbiguousID. Any message cross-linking to it
// has its SessionID cleared first.
func (s *MemoryStore) DeleteSession(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	full, err := resolveID(s.sessByID, id)
	if err != nil {
		return err
	}
	idx := s.sessByID[full]

	s.clearMessageSessionID(full)
	s.sessions = append(s.sessions[:idx], s.sessions[idx+1:]...)
	delete(s.sessByID, full)
	for i := idx; i < len(s.sessions); i++ {
		s.sessByID[s.sessions[i].ID] = i
	}
	return nil
}

// ClearSessions removes every stored session. Every message's SessionID
// cross-link is cleared first.
func (s *MemoryStore) ClearSessions(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, msg := range s.messages {
		msg.SessionID = ""
	}
	s.sessions = nil
	s.sessByID = make(map[string]int)
	return nil
}

// clearMessageSessionID blanks SessionID on any message pointing at
// sessionID, mirroring clearSessionMessageID's symmetric fix on the other
// side of the cross-link.
func (s *MemoryStore) clearMessageSessionID(sessionID string) {
	for _, msg := range s.messages {
		if msg.SessionID == sessionID {
			msg.SessionID = ""
		}
	}
}

func sessionMatchesFilter(sess *Session, filter SessionListFilter) bool {
	if filter.Status != "" && sess.Status != filter.Status {
		return false
	}
	if filter.ClientIP != "" && sess.ClientIP != filter.ClientIP {
		return false
	}
	if !filter.Since.IsZero() && sess.StartedAt.Before(filter.Since) {
		return false
	}
	if !filter.Until.IsZero() && sess.StartedAt.After(filter.Until) {
		return false
	}
	return true
}
