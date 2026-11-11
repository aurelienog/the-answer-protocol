package network_test

import (
	"errors"
	"testing"

	"the-answer-protocol/frontend/cli/internal/network"
)

func TestParserParse(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name        string
		input       string
		expected    Message
		expectError error
	}{
		{
			name:  "OK response",
			input: "OK connected",
			expected: Message{
				Type:    MessageOK,
				Payload: "connected",
				Raw:     "OK connected",
			},
		},
		{
			name:  "OK JSON response",
			input: `OK {"hp":100}`,
			expected: Message{
				Type:    MessageOK,
				Payload: `{"hp":100}`,
				Raw:     `OK {"hp":100}`,
			},
		},
		{
			name:  "ERR response",
			input: "ERR 404 ITEM_NOT_FOUND",
			expected: Message{
				Type:    MessageERR,
				Payload: "404 ITEM_NOT_FOUND",
				Raw:     "ERR 404 ITEM_NOT_FOUND",
			},
		},
		{
			name:  "EVT response",
			input: "EVT GLOBAL_CHAT Alice: Hello!",
			expected: Message{
				Type:    MessageEVT,
				Payload: "GLOBAL_CHAT Alice: Hello!",
				Raw:     "EVT GLOBAL_CHAT Alice: Hello!",
			},
		},
		{
			name:        "Empty message",
			input:       "",
			expectError: ErrEmptyMessage,
		},
		{
			name:        "Unknown message type",
			input:       "HELLO world",
			expectError: ErrUnknownMessageType,
		},
		{
			name:        "Malformed ERR",
			input:       "ERR 40 ITEM_NOT_FOUND",
			expectError: ErrMalformedMessage,
		},
		{
			name:        "ERR without name",
			input:       "ERR 404",
			expectError: ErrMalformedMessage,
		},
		{
			name:        "EVT without payload",
			input:       "EVT",
			expectError: ErrMalformedMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.input)

			if tt.expectError != nil {
				if !errors.Is(err, tt.expectError) {
					t.Fatalf(
						"expected error %v, got %v",
						tt.expectError,
						err,
					)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.expected {
				t.Errorf(
					"expected %+v, got %+v",
					tt.expected,
					got,
				)
			}
		})
	}
}