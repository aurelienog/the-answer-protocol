package server

import (
	"bufio"
	"net"
)

const maxLineBytes = 1024

// ReadEvent represents either a received line or a read error.
type ReadEvent struct {
	Line string
	Err  error
}

// ReadPump reads TCP lines and forwards them to the session handler.
// It stops when the connection closes or stop is signalled.
func ReadPump(
	conn net.Conn,
	events chan<- ReadEvent,
	stop <-chan struct{},
) {
	defer close(events)

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 256), maxLineBytes)

	for scanner.Scan() {
		event := ReadEvent{
			Line: scanner.Text(),
		}

		select {
		case events <- event:
		case <-stop:
			return
		}
	}

	if err := scanner.Err(); err != nil {
		select {
		case events <- ReadEvent{Err: err}:
		case <-stop:
		}
	}
}