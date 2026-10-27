package server

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxLineBytes = 1024

// ReadEvent represents either a received line or a read error.
type ReadEvent struct {
	Line string
	Err  error
}

// ReadPump reads TCP lines and forwards them to the session handler.
// It stops when the connection closes, a read error occurs, or stop is signalled.
func ReadPump(
	connection net.Conn,
	events chan<- ReadEvent,
	stop <-chan struct{},
) {
	defer close(events)

	// Allow room for the maximum payload plus CRLF.
	scanner := bufio.NewScanner(connection)
	scanner.Buffer(make([]byte, 256), maxLineBytes+2)
	scanner.Split(splitLF)

	for scanner.Scan() {
		line := scanner.Text()

		if len([]byte(line)) > maxLineBytes {
			sendReadError(events, stop, fmt.Errorf(
				"message exceeds maximum length of %d bytes",
				maxLineBytes,
			))
			return
		}

		if !utf8.ValidString(line) {
			sendReadError(events, stop, errors.New(
				"message is not valid UTF-8",
			))
			return
		}

		if strings.IndexFunc(line, unicode.IsControl) >= 0 {
			sendReadError(events, stop, errors.New(
				"message contains control characters",
			))
			return
		}

		select {
		case events <- ReadEvent{Line: line}:
		case <-stop:
			return
		}
	}

	if err := scanner.Err(); err != nil {
		sendReadError(events, stop, fmt.Errorf("read TCP line: %w", err))
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

func splitLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if newlineIndex := bytes.IndexByte(data, '\n'); newlineIndex >= 0 {
		line := data[:newlineIndex]

		// Accept CRLF as well as LF.
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}

		return newlineIndex + 1, line, nil
	}

	if atEOF && len(data) > 0 {
		return 0, nil, errors.New("unterminated message: expected LF before EOF")
	}

	return 0, nil, nil
}
