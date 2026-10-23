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

	"the-answer-protocol/internal/config"
)

const (
	connectTimeout = 10 * time.Second
	writeTimeout   = 10 * time.Second
)

func main() {
	cfg := config.LoadConfig()
	addr := cfg.ServerAddr()

	log.Printf("Connecting to TAP Game Server at %s...", addr)

	conn, err := net.DialTimeout("tcp", addr, connectTimeout)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	inputReader := bufio.NewReader(os.Stdin)

	// Read the server greeting.
	greeting, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to receive server greeting: %v", err)
	}

	greeting = strings.TrimSpace(greeting)
	fmt.Println(greeting)

	if greeting != "OK hello proto=1" {
		log.Fatalf("Unexpected server greeting: %s", greeting)
	}

	// Request a username before authenticating.
	username := promptUsername(inputReader)

	// Send the CONNECT command.
	if err := sendCommand(conn, "CONNECT "+username); err != nil {
		log.Fatalf("Failed to send CONNECT command: %v", err)
	}

	// Wait for the server to confirm the connection.
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to receive CONNECT response: %v", err)
	}

	response = strings.TrimSpace(response)
	fmt.Println(response)

	if response != "OK connected" {
		log.Fatalf("Could not connect to the game: %s", response)
	}

	log.Printf("Connected as %s. Type your commands below.", username)

	// Read subsequent server messages asynchronously.
	done := make(chan struct{})
	var doneOnce sync.Once

	signalDone := func() {
		doneOnce.Do(func() {
			close(done)
		})
	}

	go readServerResponses(reader, signalDone)

	// Main goroutine handles keyboard input.
	for {
		select {
		case <-done:
			log.Println("Disconnected from the game server.")
			return
		default:
		}

		fmt.Print("> ")

		input, err := inputReader.ReadString('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("Failed to read user input: %v", err)
			}
			return
		}

		command := strings.TrimSpace(input)
		if command == "" {
			continue
		}

		if err := sendCommand(conn, command); err != nil {
			log.Printf("Failed to send command: %v", err)
			return
		}

		if strings.EqualFold(command, "QUIT") {
			// The response reader will stop on "OK bye"
			// or when the server closes the connection.
			<-done
			log.Println("Disconnected from the game server.")
			return
		}
	}
}

// promptUsername asks for a valid username accepted by the server.
func promptUsername(reader *bufio.Reader) string {
	for {
		fmt.Print("Enter your username: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Failed to read username: %v", err)
		}

		username := strings.TrimSpace(input)
		if validUsername(username) {
			return username
		}

		fmt.Println(
			"Invalid username. Use 1-24 characters: letters, numbers, or underscores.",
		)
	}
}

// validUsername mirrors the username rules