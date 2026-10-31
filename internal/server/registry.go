package server

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"the-answer-protocol/internal/protocol"
)

var (
	ErrNameInUse      = errors.New("player name already in use")
	ErrInvalidSession = errors.New("invalid session")
	ErrSessionIDInUse = errors.New("session ID already in use")
)

// Registry manages authenticated player sessions safely.
type Registry struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	names    map[string]string // normalized name -> session ID
}

// NewRegistry initializes the session registry.
func NewRegistry() *Registry {
	return &Registry{
		sessions: make(map[string]*Session),
		names:    make(map[string]string),
	}
}

// Add registers an authenticated session.
//
// The player's name must already be assigned.
// Both the session ID and player name must be unique.
func (r *Registry) Add(s *Session) error {
	if r == nil || s == nil || s.ID == "" ||
		s.PlayerName == "" || s.Outbox == nil {
		return ErrInvalidSession
	}

	nameKey := normalizeName(s.PlayerName)

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sessions[s.ID]; exists {
		return ErrSessionIDInUse
	}

	if _, exists := r.names[nameKey]; exists {
		return ErrNameInUse
	}

	r.sessions[s.ID] = s
	r.names[nameKey] = s.ID

	return nil
}

func (r *Registry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, exists := r.sessions[id]
	if !exists {
		return
	}

	delete(r.sessions, id)
	delete(r.names, normalizeName(s.PlayerName))
}

// Get retrieves a session by its ID.
func (r *Registry) Get(id string) (*Session, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, exists := r.sessions[id]
	return s, exists
}

// GetByName retrieves a session by its case-insensitive player name.
func (r *Registry) GetByName(name string) (*Session, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, exists := r.names[normalizeName(name)]
	if !exists {
		return nil, false
	}

	s, exists := r.sessions[id]
	return s, exists
}

// Broadcast encodes a TAP message and queues it for every registered player.
//
// A full Outbox causes the message to be dropped for that player rather
// than blocking the entire broadcast.
func (r *Registry) Broadcast(message protocol.Message) {
	data, err := protocol.Encode(message)
	if err != nil {
		log.Printf("Failed to encode broadcast message: %v", err)
		return
	}

	// Keep the read lock while queueing messages. Remove must acquire
	// the write lock, so it cannot finish while a broadcast is enqueuing.
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, s := range r.sessions {
		if err := s.Enqueue(string(data)); err != nil {
			log.Printf(
				"Dropping TAP broadcast for %q: %v",
				s.PlayerName,
				err,
			)
		}
	}
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// Queue queues a TAP message for a specific registered session.
func (r *Registry) Queue(sessionID string, message protocol.Message) error {
	data, err := protocol.Encode(message)
	if err != nil {
		return fmt.Errorf("encode TAP message: %w", err)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	s, exists := r.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %q is not registered", sessionID)
	}

	if err := s.Enqueue(string(data)); err != nil {
		return fmt.Errorf(
			"queue message for player %q: %w",
			s.PlayerName,
			err,
		)
	}

	return nil
}