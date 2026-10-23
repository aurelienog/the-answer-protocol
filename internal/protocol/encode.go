package protocol

import (
	"bytes"
	"fmt"
	"strings"
)

// Encode serializa un mensaje TAP y añade el terminador LF.
func Encode(message Message) ([]byte, error) {
	var line string

	switch message.Kind {
	case KindOK:
		if err := validateField(message.Payload); err != nil {
			return nil, fmt.Errorf("invalid OK payload: %w", err)
		}

		if message.Payload == "" {
			line = "OK"
		} else {
			line = "OK " + message.Payload
		}

	case KindERR:
		if message.Code < 100 || message.Code > 999 {
			return nil, fmt.Errorf("invalid TAP error code: %d", message.Code)
		}

		if err := validateField(message.Name); err != nil {
			return nil, fmt.Errorf("invalid TAP error name: %w", err)
		}

		if message.Name == "" {
			return nil, fmt.Errorf("TAP error name cannot be empty")
		}

		line = fmt.Sprintf("ERR %03d %s", message.Code, message.Name)

	case KindEVT:
		if err := validateField(message.EventType); err != nil {
			return nil, fmt.Errorf("invalid TAP event type: %w", err)
		}

		if err := validateField(message.Data); err != nil {
			return nil, fmt.Errorf("invalid TAP event data: %w", err)
		}

		if strings.TrimSpace(message.EventType) == "" {
			return nil, fmt.Errorf("TAP event type cannot be empty")
		}

		if message.Data == "" {
			line = "EVT " + message.EventType
		} else {
			line = "EVT " + message.EventType + " " + message.Data
		}

	default:
		return nil, fmt.Errorf("unknown TAP message kind: %d", message.Kind)
	}

	var buffer bytes.Buffer
	buffer.WriteString(line)
	buffer.WriteByte('\n')

	return buffer.Bytes(), nil
}

// validateField evita que un campo introduzca mensajes adicionales
// mediante saltos de línea.
func validateField(value string) error {
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("CR and LF are not allowed inside a TAP message")
	}

	return nil
}