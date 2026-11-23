package server

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"the-answer-protocol/internal/protocol"
)

const (
	OutboxSize   = 32
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
func WritePump(session *Session, done chan<- struct{}, logger *slog.Logger) {
	if session == nil {
		close(done)
		return
	}
	if logger == nil {
		panic("write pump requires a logger")
	}
	defer close(done)

	for message := range session.Outbox {
		if err := session.Connection.SetWriteDeadline(
			time.Now().Add(WriteTimeout),
		); err != nil {
			logger.Error("session.write_deadline_failed",
				slog.String("session_id", session.ID),
				slog.String("error", err.Error()),
			)
			_ = session.Connection.Close()
			return
		}

		if _, err := fmt.Fprint(session.Connection, message); err != nil {
			logger.Error("session.write_failed",
				slog.String("session_id", session.ID),
				slog.String("error", err.Error()),
			)
			_ = session.Connection.Close()
			return
		}
		fields := strings.Fields(message)
		attrs := []any{
			slog.String("session_id", session.ID),
			slog.String("remote_ip", session.RemoteIP),
			slog.String("player", session.GetPlayerName()),
			slog.Int("bytes", len(message)),
		}
		if len(fields) > 0 {
			attrs = append(attrs, slog.String("response_kind", strings.TrimSpace(fields[0])))
		}
		if len(fields) >= 1 && fields[0] == "ERR" {
			if len(fields) >= 2 {
				attrs = append(attrs, slog.String("error_code", fields[1]))
			}
			if len(fields) >= 3 {
				attrs = append(attrs, slog.String("error_name", fields[2]))
			}
			logger.Warn("session.response_sent", attrs...)
		} else {
			logger.Info("session.response_sent", attrs...)
		}
	}
}
