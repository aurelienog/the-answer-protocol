package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"the-answer-protocol/frontend/cli/network"
	"the-answer-protocol/frontend/cli/ui"
	"the-answer-protocol/internal/config"
	"the-answer-protocol/internal/protocol"
)

func main() {
	ui.PrintBanner()

	cfg := config.LoadConfig()
	addr := cfg.ServerAddr()

	ctx := context.Background()

	ui.PrintInfo("Connecting to TAP Game Server...")
	ui.PrintServerStatus(addr, ui.StatusConnecting)

	// Establish TCP connection and validate server greeting.
	connection, err := network.Dial(ctx, addr)
	if err != nil {
		ui.PrintError("Failed to connect to server.")
		log.Printf("Connection error: %v", err)
		return
	}

	// Create the network client.
	client, err := network.NewClient(ctx, connection)
	if err != nil {
		ui.PrintError("Failed to initialize network client.")
		log.Printf("Client initialization error: %v", err)
		return
	}
	defer client.Close()

	ui.PrintServerStatus(addr, ui.StatusConnected)

	inputReader := bufio.NewReader(os.Stdin)

	// Authenticate before starting the game session.
	username, err := authenticate(client, inputReader)
	if err != nil {
		ui.PrintError("Authentication failed.")
		log.Printf("Authentication error: %v", err)
		return
	}

	if username == "" {
		return
	}

	ui.PrintServerStatus(addr, ui.StatusAuthenticated)
	ui.PrintInfo("Connected as " + username + ".")
	ui.PrintInfo("You can now enter game commands.")

	// Receive asynchronous server events.
	go readServerEvents(client)

	// Read and send player commands.
	for {
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

		if commandName == "CONNECT" {
			ui.PrintError("You are already authenticated.")
			continue
		}

		if commandName == "QUIT" && len(fields) != 1 {
			ui.PrintError("Usage: QUIT")
			continue
		}

		// Send the command and wait for its response.
		response, err := client.Send(ctx, command)

		if response.Kind == protocol.KindOK ||
			response.Kind == protocol.KindERR {
			ui.HandleResponse(formatMessage(response))
		}

		if err != nil {
			var serverErr *protocol.ServerError

			if errors.As(err, &serverErr) {
				continue
			}

			ui.PrintError("Connection to server lost.")
			log.Printf("Command error: %v", err)
			return
		}

		if commandName == "QUIT" {
			ui.PrintInfo("Disconnected from the game server.")
			return
		}
	}
}

// authenticate handles the initial CONNECT exchange.
func authenticate(
	client *network.Client,
	inputReader *bufio.Reader,
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

		if len(fields) != 2 ||
			!strings.EqualFold(fields[0], "CONNECT") {
			ui.PrintError(
				"You must authenticate first. Usage: CONNECT <username>.",
			)
			continue
		}

		username := fields[1]

		err = client.Authenticate(username)
		if err != nil {
			if errors.Is(err, network.ErrNameInUse) ||
				errors.Is(err, protocol.ErrInvalidUsername) {
				ui.PrintError(err.Error())
				continue
			}

			return "", err
		}

		return username, nil
	}
}

// readServerEvents handles asynchronous EVT messages.
func readServerEvents(client *network.Client) {
	for {
		select {
		case event, ok := <-client.Events():
			if !ok {
				return
			}
			ui.HandleResponse(event.Raw)

		case <-client.Done():
			return
		}
	}
}

// formatMessage reconstructs the textual representation of a response.
func formatMessage(message protocol.Message) string {
	switch message.Kind {
	case protocol.KindOK:
		if message.Payload == "" {
			return "OK"
		}
		return "OK " + message.Payload

	case protocol.KindERR:
		return fmt.Sprintf("ERR %03d %s", message.Code, message.Name)

	default:
		return ""
	}
}