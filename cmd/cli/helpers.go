package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

// sendCommand sends one line to the TAP server.
func sendCommand(conn net.Conn, command string) error {
	if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}

	if _, err := fmt.Fprintln(conn, command); err != nil {
		return fmt.Errorf("write command: %w", err)
	}

	return nil
}

// readServerResponses reads and prints server messages until the
// connection closes or the server responds with OK bye.
func readServerResponses(reader *bufio.Reader, signalDone func()) {
	defer signalDone()

	for {
		line, err := reader.ReadString('\n')

		// Print a final partial line if the connection closes
		// without a terminating newline.
		if line != "" {
			message := strings.TrimSpace(line)
			fmt.Println(message)

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