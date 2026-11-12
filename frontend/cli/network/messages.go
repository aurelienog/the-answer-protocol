package network

import "the-answer-protocol/internal/protocol"

// IncomingMessage represents a message received from the TAP server.
// Message contains the parsed protocol data.
// Raw preserves the original server message.
type IncomingMessage struct {
	Message protocol.Message
	Raw     string
}