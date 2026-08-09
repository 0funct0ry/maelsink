package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
)

// MemoryStore is an in-memory MessageStore. It is the only backend for
// M1.0; M2.0 replaces it with a SQLite-backed implementation of the same
// MessageStore interface.
type MemoryStore struct {
	mu       sync.RWMutex
	messages []*Message
	byID     map[string]int // id -> index into messages
}

// NewMemoryStore returns an empty, ready-to-use MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		byID: make(map[string]int),
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
	return true
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
	}
	return stats, nil
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
	return nil
}

// Clear removes every stored message.
func (s *MemoryStore) Clear(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = nil
	s.byID = make(map[string]int)
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
