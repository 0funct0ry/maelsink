package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

// Get returns the message with the given ID, or ErrNotFound.
func (s *MemoryStore) Get(_ context.Context, id string) (*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx, ok := s.byID[id]
	if !ok {
		return nil, ErrNotFound
	}
	return s.messages[idx], nil
}

// List returns messages newest-first, paginated by filter, plus the total
// count ignoring pagination.
func (s *MemoryStore) List(_ context.Context, filter ListFilter) ([]*Message, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.messages)

	// Build newest-first order without mutating s.messages.
	ordered := make([]*Message, total)
	for i, msg := range s.messages {
		ordered[total-1-i] = msg
	}

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

// Delete removes the message with the given ID, or returns ErrNotFound.
func (s *MemoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx, ok := s.byID[id]
	if !ok {
		return ErrNotFound
	}

	s.messages = append(s.messages[:idx], s.messages[idx+1:]...)
	delete(s.byID, id)
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
