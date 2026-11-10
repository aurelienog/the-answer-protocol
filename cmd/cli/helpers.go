package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"the-answer-protocol/frontend/cli/ui"
	"the-answer-protocol/internal/protocol"
)

// sendCommand sends one line to the TAP server with a write timeout.
func sendCommand(conn net.Conn, command string) error {
	if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}

	_, writeErr := fmt.Fprintln(conn, command)
	resetErr := conn.SetWriteDeadline(time.Time{})

	if writeErr != nil {
		return fmt.Errorf("write command: %w", writeErr)
	}

	if resetErr != nil {
		return fmt.Errorf("reset write deadline: %w", resetErr)
	}

	return nil
}

// readServerResponses reads server messages and delegates their presentation
// to the UI. It stops when the server responds with OK bye or the connection
// closes.
func readServerResponses(reader *bufio.Reader, signalDone func()) {
	defer signalDone()

	for {
		line, err := reader.ReadString('\n')

		if line != "" {
			message := strings.TrimRight(line, "\r\n")
			ui.HandleResponse(message)

			if message == "OK bye" {
				return
			}
		}

		if err != nil {
			return
		}
	}
}

// validUsername mirrors the username rules enforced by the server.
func validUsername(username string) bool {
	return protocol.ValidUsername(username)
}
