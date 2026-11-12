package protocol

import (
	"fmt"
	"strings"
)

// ParseMessage parses a raw TAP message.
func ParseMessage(raw string) (Message, error) {
	kind, payload, found := strings.Cut(raw, " ")
	if !found {
		payload = ""
	}

	switch kind {
	case "OK":
		return Message{
			Kind:    KindOK,
			Payload: payload,
		}, nil

	case "ERR":
		serverErr, err := ParseServerError(payload)
		if err != nil {
			return Message{}, err
		}

		return Message{
			Kind: KindERR,
			Code: serverErr.Code,
			Name: serverErr.Name,
		}, nil

	case "EVT":
		eventType, data, found := strings.Cut(payload, " ")
		if !found {
			data = ""
		}

		if eventType == "" {
			return Message{}, fmt.Errorf("missing TAP event type")
		}

		return Message{
			Kind:      KindEVT,
			EventType: eventType,
			Data:      data,
		}, nil

	default:
		return Message{}, fmt.Errorf(
			"unknown TAP message type: %q",
			kind,
		)
	}
}