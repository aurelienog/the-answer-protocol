package ui

import "strings"

type MessageType int

const (
	MessageUnknown MessageType = iota
	MessageOK
	MessageError
	MessageEvent
)

type ServerMessage struct {
	Type    MessageType
	Payload string
}

// ParseServerMessage classifies a complete TAP message.
func ParseServerMessage(line string) ServerMessage {
	line = strings.TrimSpace(line)

	switch {
	case line == "OK":
		return ServerMessage{Type: MessageOK}

	case strings.HasPrefix(line, "OK "):
		return ServerMessage{
			Type:    MessageOK,
			Payload: strings.TrimSpace(strings.TrimPrefix(line, "OK ")),
		}

	case strings.HasPrefix(line, "ERR "):
		return ServerMessage{
			Type:    MessageError,
			Payload: strings.TrimSpace(strings.TrimPrefix(line, "ERR ")),
		}

	case strings.HasPrefix(line, "EVT "):
		return ServerMessage{
			Type:    MessageEvent,
			Payload: strings.TrimSpace(strings.TrimPrefix(line, "EVT ")),
		}

	default:
		return ServerMessage{
			Type:    MessageUnknown,
			Payload: line,
		}
	}
}
