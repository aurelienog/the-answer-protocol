package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"the-answer-protocol/cmd/cli/ui"
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

// validUsername mirrors the username rules enforced by the server:
// 1-24 ASCII letters, digits, or underscores.
func validUsername(username string) bool {
	if len(username) < 1 || len(username) > 24 {
		return false
	}

	for _, r := range username {
		if !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_') {
			return false
		}
	}

	return true
}