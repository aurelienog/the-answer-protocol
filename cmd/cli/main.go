package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"the-answer-protocol/cmd/cli/ui"
	"the-answer-protocol/internal/config"
)

const (
	connectTimeout = 10 * time.Second
	writeTimeout   = 10 * time.Second
)

func main() {
	ui.PrintBanner()

	cfg := config.LoadConfig()
	addr := cfg.ServerAddr()

	ui.PrintInfo("Connecting to TAP Game Server...")
	ui.PrintServerStatus(addr, ui.StatusConnecting)

	conn, err := net.DialTimeout("tcp", addr, connectTimeout)
	if err != nil {
		ui.PrintError("Failed to connect to server.")
		log.Printf("Connection error: %v", err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	inputReader := bufio.NewReader(os.Stdin)

	// Read and validate the server greeting.
	greeting, err := reader.ReadString('\n')
	if err != nil {
		ui.PrintError("Failed to receive server greeting.")
		log.Printf("Greeting error: %v", err)
		return
	}

	greeting = strings.TrimSpace(greeting)

	if greeting != "OK hello proto=1" {
		ui.PrintError("Unexpected server greeting.")
		log.Printf("Received greeting: %q", greeting)
		return
	}

	ui.HandleResponse(greeting)
	ui.PrintServerStatus(addr, ui.StatusConnected)

	// Authenticate explicitly using CONNECT <username>.
	username, err := authenticate(conn, reader, inputReader, addr)
	if err != nil {
		ui.PrintError("Authentication failed.")
		log.Printf("Authentication error: %v", err)
		return
	}

	// An empty username means the user reached EOF before authenticating.
	if username == "" {
		return
	}
	ui.PrintServerStatus(addr, ui.StatusAuthenticated)
	ui.PrintInfo("Connected as " + username + ".")
	ui.PrintInfo("You can now enter game commands.")

	// Coordinate the asynchronous server reader's lifecycle.
	done := make(chan struct{})
	var doneOnce sync.Once

	signalDone := func() {
		doneOnce.Do(func() {
			close(done)
		})
	}

	go readServerResponses(reader, signalDone)

	// The main goroutine manages command validation and user input.
	for {
		select {
		case <-done:
			ui.PrintInfo("Disconnected from the game server.")
			return
		default:
		}

		ui.PrintPrompt()

		input, err := inputReader.ReadString('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				ui.PrintError("Failed to read user input.")
				log.Printf("Input error: %v", err)
			}
			return
		}

		command := strings.TrimSpace(input)
		if command == "" {
			continue
		}

		fields := strings.Fields(command)
		commandName := strings.ToUpper(fields[0])

		// CONNECT is only valid before authentication.
		if commandName == "CONNECT" {
			ui.PrintError("You are already authenticated.")
			continue
		}

		// Validate QUIT and other commands before sending.
		if commandName == "QUIT" && len(fields) != 1 {
			ui.PrintError("Usage: QUIT")
			continue
		}

		// Send the command to the server.
		if err := sendCommand(conn, command); err != nil {
			ui.PrintError("Failed to send command to server.")
			log.Printf("Command send error: %v", err)
			return
		}

		if commandName == "QUIT" {
			ui.PrintInfo("Waiting for the server to close the session...")

			// Wait until the server reader detects the connection closure.
			<-done

			ui.PrintInfo("Disconnected from the game server.")
			return
		}
	}
}

// authenticate waits for CONNECT <username> and returns only after
// the server confirms successful authentication.
func authenticate(
	conn net.Conn,
	reader *bufio.Reader,
	inputReader *bufio.Reader,
	address string,
) (string, error) {
	for {
		ui.PrintInfo("Authenticate using CONNECT <username>.")
		ui.PrintPrompt()

		input, err := inputReader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", nil
			}
			return "", fmt.Errorf("read CONNECT command: %w", err)
		}

		command := strings.TrimSpace(input)
		if command == "" {
			continue
		}

		fields := strings.Fields(command)

		if strings.EqualFold(fields[0], "QUIT") {
			if len(fields) != 1 {
				ui.PrintError("Usage: QUIT")
				continue
			}

			return "", nil
		}

		// Only CONNECT with exactly one username is allowed.
		if len(fields) != 2 || !strings.EqualFold(fields[0], "CONNECT") {
			ui.PrintError("You must authenticate first. Usage: CONNECT <username>.")
			continue
		}

		username := fields[1]
		if !validUsername(username) {
			ui.PrintError(
				"Invalid username. Use 1-24 characters: letters, numbers, or underscores.",
			)
			continue
		}

		if err := sendCommand(conn, command); err != nil {
			return "", fmt.Errorf("send CONNECT command: %w", err)
		}

		response, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("read CONNECT response: %w", err)
		}

		response = strings.TrimSpace(response)
		message := ui.ParseServerMessage(response)

		ui.HandleResponse(response)

		// Authentication succeeds only when confirmed by the server.
		if message.Type == ui.MessageOK &&
			strings.EqualFold(message.Payload, "connected") {
			return username, nil
		}
		// On an authentication error, let the user try another username.
	}
}
