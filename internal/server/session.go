package server

import (
	"errors"
	"sync"
	"net"
)

var (
	ErrOutboxClosed = errors.New("session Outbox is closed")
	ErrOutboxFull   = errors.New("session Outbox is full")
)

// Session represents an active player network pipeline context
type Session struct {
	ID         string   // Unique identifier (usually RemoteAddr or custom GUID)
	Connection net.Conn // The active open TCP socket connection
	PlayerName string   // Bound player entity name within the game world
	CurrentRoom string  // ID of the room the player is currently occupying

	Outbox chan string
	outboxMu     sync.Mutex
	outboxClosed bool
}

// Enqueue queues a message without blocking.
// It is safe to call concurrently with CloseOutbox.
func (s *Session) Enqueue(message string) error {
	if s == nil {
		return ErrInvalidSession
	}

	s.outboxMu.Lock()
	defer s.outboxMu.Unlock()

	if s.outboxClosed || s.Outbox == nil {
		return ErrOutboxClosed
	}

	select {
	case s.Outbox <- message:
		return nil
	default:
		return ErrOutboxFull
	}
}

// CloseOutbox closes the output channel exactly once.
// It is safe to call concurrently with Enqueue.
func (s *Session) CloseOutbox() {
	if s == nil {
		return
	}

	s.outboxMu.Lock()
	defer s.outboxMu.Unlock()

	if s.outboxClosed {
		return
	}

	s.outboxClosed = true

	if s.Outbox != nil {
		close(s.Outbox)
	}
}