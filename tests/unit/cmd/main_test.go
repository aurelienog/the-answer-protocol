package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/server"
	"the-answer-protocol/internal/world"
)

const testTimeout = 3 * time.Second

func createTestWorld(t *testing.T) *world.World {
	t.Helper()

	worldData := struct {
		Rooms map[string]world.Room `json:"rooms"`
	}{
		Rooms: map[string]world.Room{
			initialRoomID: {
				ID:          initialRoomID,
				Name:        "Harbor Square",
				Description: "A small harbor square.",
				Exits:       map[string]string{},
				Items:       []string{},
				NPCs:        []string{},
			},
		},
	}

	data, err := json.Marshal(worldData)
	if err != nil {
		t.Fatalf("marshal test world: %v", err)
	}

	worldFile := filepath.Join(t.TempDir(), "world.json")
	if err := os.WriteFile(worldFile, data, 0600); err != nil {
		t.Fatalf("write test world: %v", err)
	}

	gameWorld, err := world.Load(worldFile)
	if err != nil {
		t.Fatalf("load test world: %v", err)
	}

	return gameWorld
}

func startTestConnection(
	t *testing.T,
) (net.Conn, <-chan struct{}, *server.Registry, *world.World) {
	t.Helper()

	serverConn, clientConn := net.Pipe()

	registry := server.NewRegistry()
	gameWorld := createTestWorld(t)

	dispatcher := commands.NewDispatcher()
	commands.RegisterCommands(dispatcher)

	handlerDone := make(chan struct{})

	go func() {
		defer close(handlerDone)
		handleConnection(serverConn, dispatcher, registry, gameWorld)
	}()

	if err := clientConn.SetDeadline(time.Now().Add(testTimeout)); err != nil {
		clientConn.Close()
		t.Fatalf("set client deadline: %v", err)
	}

	reader := bufio.NewReader(clientConn)

	greeting, err := reader.ReadString('\n')
	if err != nil {
		clientConn.Close()
		t.Fatalf("read greeting: %v", err)
	}

	if greeting != "OK hello proto=1\n" {
		clientConn.Close()
		t.Fatalf("unexpected greeting: %q", greeting)
	}

	t.Cleanup(func() {
		_ = clientConn.Close()

		select {
		case <-handlerDone:
		case <-time.After(testTimeout):
			t.Error("handleConnection did not terminate before timeout")
		}
	})

	return clientConn, handlerDone, registry, gameWorld
}

func readTestMessage(t *testing.T, reader *bufio.Reader) string {
	t.Helper()

	if err := readerWriteDeadlinePlaceholder(); err != nil {
		t.Fatalf("prepare message read: %v", err)
	}

	message, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read server message: %v", err)
	}

	return strings.TrimSuffix(message, "\n")
}

// TestHandleConnectionConnectAndQuit verifies the normal connection lifecycle.
func TestHandleConnectionConnectAndQuit(t *testing.T) {
	clientConn, handlerDone, registry, gameWorld := startTestConnection(t)
	reader := bufio.NewReader(clientConn)

	if _, err := fmt.Fprint(clientConn, "CONNECT Alice\n"); err != nil {
		t.Fatalf("send CONNECT: %v", err)
	}

	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read CONNECT response: %v", err)
	}

	if response != "OK connected\n" {
		t.Fatalf("unexpected CONNECT response: %q", response)
	}

	event, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read PLAYER_ENTER event: %v", err)
	}

	if event != "EVT PLAYER_ENTER Alice\n" {
		t.Fatalf("unexpected PLAYER_ENTER event: %q", event)
	}

	if _, exists := registry.GetByName("Alice"); !exists {
		t.Fatal("expected Alice to be present in the session registry")
	}

	roomID, err := gameWorld.GetPlayerRoom("Alice")
	if err != nil {
		t.Fatalf("expected Alice to be present in the world: %v", err)
	}

	if roomID != initialRoomID {
		t.Fatalf("expected Alice in room %q, got %q", initialRoomID, roomID)
	}

	if _, err := fmt.Fprint(clientConn, "QUIT\n"); err != nil {
		t.Fatalf("send QUIT: %v", err)
	}

	quitResponse, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read QUIT response: %v", err)
	}

	if quitResponse != "OK bye\n" {
		t.Fatalf("unexpected QUIT response: %q", quitResponse)
	}

	select {
	case <-handlerDone:
	case <-time.After(testTimeout):
		t.Fatal("handleConnection did not terminate after QUIT")
	}

	if _, exists := registry.GetByName("Alice"); exists {
		t.Fatal("expected Alice to be removed from the session registry")
	}

	if _, err := gameWorld.GetPlayerRoom("Alice"); err == nil {
		t.Fatal("expected Alice to be removed from the world")
	}
}
