package network_test

import (
	"errors"
	"io"
	"net"
	"testing"

	"the-answer-protocol/frontend/cli/network"
)

func TestReadLine(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{
			name:  "normal LF",
			input: "OK connected\n",
			want:  "OK connected",
		},
		{
			name:  "normal CRLF",
			input: "OK connected\r\n",
			want:  "OK connected",
		},
		{
			name:  "empty line",
			input: "\n",
			want:  "",
		},
		{
			name:    "too long",
			input:   string(make([]byte, network.MaxLineBytes+1)) + "\n",
			wantErr: network.ErrLineTooLong,
		},
		{
			name:    "invalid UTF-8",
			input:   string([]byte{0xff, '\n'}),
			wantErr: network.ErrInvalidUTF8,
		},
		{
			name:    "control character",
			input:   "OK\tconnected\n",
			wantErr: network.ErrControlChar,
		},
		{
			name:    "unterminated message",
			input:   "OK connected",
			wantErr: network.ErrUnterminatedLine,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, client := net.Pipe()
			defer server.Close()
			defer client.Close()

			transport := network.NewTransport(client)

			go func() {
				_, _ = io.WriteString(server, tt.input)
				_ = server.Close()
			}()

			got, err := transport.ReadLine()

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestReadLineMaximumLength(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	transport := network.NewTransport(client)

	message := make([]byte, network.MaxLineBytes)
	for i := range message {
		message[i] = 'A'
	}

	go func() {
		_, _ = server.Write(append(message, '\n'))
	}()

	got, err := transport.ReadLine()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != network.MaxLineBytes {
		t.Fatalf("expected %d bytes, got %d", network.MaxLineBytes, len(got))
	}
}

func TestReadLineContinuesAfterOversizedMessage(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	transport := network.NewTransport(client)

	oversized := make([]byte, network.MaxLineBytes+10)
	for i := range oversized {
		oversized[i] = 'A'
	}

	go func() {
		_, _ = server.Write(append(oversized, '\n'))
		_, _ = io.WriteString(server, "OK connected\n")
	}()

	_, err := transport.ReadLine()

	if !errors.Is(err, network.ErrLineTooLong) {
		t.Fatalf("expected ErrLineTooLong, got %v", err)
	}

	got, err := transport.ReadLine()
	if err != nil {
		t.Fatalf("unexpected error reading next message: %v", err)
	}

	if got != "OK connected" {
		t.Fatalf("expected OK connected, got %q", got)
	}
}

func TestWriteLine(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	transport := network.NewTransport(client)

	go func() {
		buffer := make([]byte, len("LOOK\n"))
		_, _ = io.ReadFull(server, buffer)

		if string(buffer) != "LOOK\n" {
			t.Errorf("expected LOOK\\n, got %q", string(buffer))
		}
	}()

	if err := transport.WriteLine("LOOK"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteLineRejectsInvalidMessages(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	transport := network.NewTransport(client)

	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "newline",
			message: "LOOK\nQUIT",
		},
		{
			name:    "carriage return",
			message: "LOOK\rQUIT",
		},
		{
			name:    "too long",
			message: string(make([]byte, network.MaxLineBytes+1)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := transport.WriteLine(tt.message)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
