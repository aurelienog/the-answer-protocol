package server

import (
	"fmt"
	"log"
	"time"

	"the-answer-protocol/internal/protocol"
)

const (
	OutboxSize  = 32
	WriteTimeout = 10 * time.Second
)

// SendMessage encodes a TAP message and queues it for WritePump.
func SendMessage(session *Session, message protocol.Message) error {
	if session == nil {
		return ErrInvalidSession
	}

	data, err := protocol.Encode(message)
	if err != nil {
		return fmt.Errorf("encode TAP message: %w", err)
	}

	if err := session.Enqueue(string(data)); err != nil {
		return fmt.Errorf("queue message for session %s: %w", session.ID, err)
	}

	return nil
}

// WritePump is the only goroutine that writes to the session connection.
// It drains the output queue until the queue is closed or a write fails.
func WritePump(session *Session, done chan<- struct{}) {
	defer close(done)

	for message := range session.Outbox {
		if err := session.Connection.SetWriteDeadline(
			time.Now().Add(WriteTimeout),
		); err != nil {
			log.Printf(
				"Failed to set write deadline for %s: %v",
				session.ID,
				err,
			)
			_ = session.Connection.Close()
			return
		}

		if _, err := fmt.Fprint(session.Connection, message); err != nil {
			log.Printf("Write error for session %s: %v", session.ID, err)
			_ = session.Connection.Close()
			return
		}
	}
}