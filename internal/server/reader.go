package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxLineBytes = 1024

// ReadEvent represents either a received line or a read error.
type ReadEvent struct {
	Line          string
	Err           error
	ProtocolError error
}

// ReadPump reads TCP lines and forwards them to the session handler.
// It stops when the connection closes, a read error occurs, or stop is signalled.
func ReadPump(
	connection net.Conn,
	events chan<- ReadEvent,
	stop <-chan struct{},
) {
	defer close(events)

	reader := bufio.NewReader(connection)
	for {
		line, terminated, tooLong, err := readBoundedLine(reader, maxLineBytes)
		if tooLong {
			sendProtocolError(events, stop, fmt.Errorf(
				"message exceeds maximum length of %d bytes",
				maxLineBytes,
			))
			if err != nil || !terminated {
				if err != nil && !errors.Is(err, io.EOF) {
					sendReadError(events, stop, fmt.Errorf("read TCP line: %w", err))
				}
				return
			}
			continue
		}

		if err != nil {
			if errors.Is(err, io.EOF) && len(line) == 0 {
				return
			}
			if errors.Is(err, io.EOF) {
				sendProtocolError(events, stop, errors.New("unterminated message: expected LF before EOF"))
				return
			}
			sendReadError(events, stop, fmt.Errorf("read TCP line: %w", err))
			return
		}
		if !terminated {
			sendProtocolError(events, stop, errors.New("unterminated message: expected LF before EOF"))
			return
		}

		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		if len(line) > maxLineBytes {
			sendProtocolError(events, stop, fmt.Errorf(
				"message exceeds maximum length of %d bytes",
				maxLineBytes,
			))
			continue
		}

		message := string(line)
		switch {
		case !utf8.ValidString(message):
			sendProtocolError(events, stop, errors.New("message is not valid UTF-8"))
		case strings.IndexFunc(message, unicode.IsControl) >= 0:
			sendProtocolError(events, stop, errors.New("message contains control characters"))
		default:
			select {
			case events <- ReadEvent{Line: message}:
			case <-stop:
				return
			}
		}
	}
}

func readBoundedLine(
	reader *bufio.Reader,
	maxBytes int,
) ([]byte, bool, bool, error) {
	line := make([]byte, 0, maxBytes+1)
	tooLong := false
	for {
		chunk, err := reader.ReadSlice('\n')
		terminated := len(chunk) > 0 && chunk[len(chunk)-1] == '\n'
		if terminated {
			chunk = chunk[:len(chunk)-1]
		}
		if !tooLong {
			if len(line)+len(chunk) > maxBytes+1 {
				tooLong = true
				line = nil
			} else {
				line = append(line, chunk...)
			}
		}

		if err == nil {
			return line, terminated, tooLong, nil
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		return line, false, tooLong, err
	}
}

func sendProtocolError(
	events chan<- ReadEvent,
	stop <-chan struct{},
	readError error,
) {
	select {
	case events <- ReadEvent{ProtocolError: readError}:
	case <-stop:
	}
}

func sendReadError(
	events chan<- ReadEvent,
	stop <-chan struct{},
	readError error,
) {
	select {
	case events <- ReadEvent{Err: readError}:
	case <-stop:
	}
}
