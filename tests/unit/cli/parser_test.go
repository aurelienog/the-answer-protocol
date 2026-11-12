package network_test

import (
	"testing"

	"the-answer-protocol/internal/protocol"
)

func TestParseMessage(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    protocol.Message
		expectError bool
	}{
		{
			name:  "OK response",
			input: "OK connected",
			expected: protocol.Message{
				Kind:    protocol.KindOK,
				Payload: "connected",
			},
		},
		{
			name:  "OK JSON response",
			input: `OK {"hp":100}`,
			expected: protocol.Message{
				Kind:    protocol.KindOK,
				Payload: `{"hp":100}`,
			},
		},
		{
			name:  "ERR response",
			input: "ERR 404 ITEM_NOT_FOUND",
			expected: protocol.Message{
				Kind: protocol.KindERR,
				Code: 404,
				Name: "ITEM_NOT_FOUND",
			},
		},
		{
			name:  "EVT response",
			input: "EVT GLOBAL_CHAT Alice: Hello!",
			expected: protocol.Message{
				Kind:      protocol.KindEVT,
				EventType: "GLOBAL_CHAT",
				Data:      "Alice: Hello!",
			},
		},
		{name: "empty message", input: "", expectError: true},
		{name: "unknown message type", input: "HELLO world", expectError: true},
		{name: "malformed ERR code", input: "ERR 40 ITEM_NOT_FOUND", expectError: true},
		{name: "ERR without name", input: "ERR 404", expectError: true},
		{name: "EVT without event type", input: "EVT", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := protocol.ParseMessage(tt.input)
			if tt.expectError {
				if err == nil {
					t.Fatal("ParseMessage() succeeded, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseMessage() error = %v", err)
			}
			if got != tt.expected {
				t.Errorf("ParseMessage() = %+v, want %+v", got, tt.expected)
			}
		})
	}
}
