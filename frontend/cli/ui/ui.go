package ui

import (
	"fmt"
	"strings"
)

const (
	width = 54

	StatusDisconnected  = "Disconnected"
	StatusConnecting    = "Connecting..."
	StatusConnected     = "Connected"
	StatusAuthenticated = "Authenticated"
)

// PrintServerStatus displays the server address and connection status.
func PrintServerStatus(address, status string) {
	fmt.Println(strings.Repeat("-", width))
	fmt.Printf("  Server: %s\n", address)
	fmt.Printf("  Status: %s\n", status)
	fmt.Println(strings.Repeat("-", width))
}

// PrintPrompt displays the command input prompt.
func PrintPrompt() {
	fmt.Print("TAP> ")
}

// PrintSeparator displays a visual separator.
func PrintSeparator() {
	fmt.Println(strings.Repeat("-", width))
}

// HandleResponse classifies and delegates the presentation of TAP messages.
func HandleResponse(response string) {
	message := ParseServerMessage(response)

	switch message.Type {
	case MessageOK:
		handleOKResponse(message.Payload)

	case MessageError:
		PrintError(message.Payload)

	case MessageEvent:
		PrintEvent(message.Payload)

	default:
		PrintMessage(message.Payload)
	}
}
