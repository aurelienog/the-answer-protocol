package internal_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

func createCommandTestWorld(t *testing.T) *world.World {
	t.Helper()

	worldData := `{
		"rooms": {
			"harbor_square": {
				"id": "harbor_square",
				"name": "Harbor Square",
				"description": "A small harbor square.",
				"exits": {
					"north": "market"
				},
				"items": ["map"],
				"npcs": ["guard"]
			},
			"market": {
				"id": "market",
				"name": "Market",
				"description": "A busy market.",
				"exits": {
					"south": "harbor_square"
				},
				"items": [],
				"npcs": []
			}
		}
	}`

	worldFile := filepath.Join(t.TempDir(), "world.json")

	if err := os.WriteFile(worldFile, []byte(worldData), 0600); err != nil {
		t.Fatalf("write test world: %v", err)
	}

	gameWorld, err := world.Load(worldFile)
	if err != nil {
		t.Fatalf("load test world: %v", err)
	}

	return gameWorld
}

func addTestPlayer(t *testing.T, gameWorld *world.World, name string) {
	t.Helper()

	if err := gameWorld.AddPlayer(name, "harbor_square"); err != nil {
		t.Fatalf("add test player: %v", err)
	}
}

// TestLookCommandReturnsCurrentRoom checks the room and its contents.
func TestLookCommandReturnsCurrentRoom(t *testing.T) {
	gameWorld := createCommandTestWorld(t)
	addTestPlayer(t, gameWorld, "Alice")

	command := commands.NewLookCommand()

	message, err := command.Execute(&commands.Context{
		PlayerName: "Alice",
		World:      gameWorld,
	})
	if err != nil {
		t.Fatalf("LOOK returned unexpected error: %v", err)
	}

	if message.Kind != protocol.KindOK {
		t.Fatalf("expected OK message, got kind %v", message.Kind)
	}

	var response struct {
		Room struct {
			ID string `json:"id"`
		} `json:"room"`
		Players []string `json:"players"`
		Items   []string `json:"items"`
		NPCs    []string `json:"npcs"`
	}

	if err := json.Unmarshal([]byte(message.Payload), &response); err != nil {
		t.Fatalf("decode LOOK response: %v; payload: %q", err, message.Payload)
	}

	if response.Room.ID != "harbor_square" {
		t.Errorf("expected room harbor_square, got %q", response.Room.ID)
	}

	if len(response.Players) != 1 || response.Players[0] != "Alice" {
		t.Errorf("expected Alice in room players, got %v", response.Players)
	}

	if len(response.Items) != 1 || response.Items[0] != "map" {
		t.Errorf("expected map in room items, got %v", response.Items)
	}

	if len(response.NPCs) != 1 || response.NPCs[0] != "guard" {
		t.Errorf("expected guard in room NPCs, got %v", response.NPCs)
	}
}

// TestLookCommandRejectsNilWorld checks missing world configuration.
func TestLookCommandRejectsNilWorld(t *testing.T) {
	command := commands.NewLookCommand()

	_, err := command.Execute(&commands.Context{
		PlayerName: "Alice",
		World:      nil,
	})

	if err == nil {
		t.Fatal("expected an error when world is nil")
	}
}

// TestMoveCommandChangesRoom checks successful movement.
func TestMoveCommandChangesRoom(t *testing.T) {
	gameWorld := createCommandTestWorld(t)
	addTestPlayer(t, gameWorld, "Alice")

	command := commands.NewMoveCommand()

	message, err := command.Execute(&commands.Context{
		PlayerName: "Alice",
		World:      gameWorld,
		RawArgs:    []string{"north"},
	})
	if err != nil {
		t.Fatalf("MOVE returned unexpected error: %v", err)
	}

	if message.Kind != protocol.KindOK {
		t.Fatalf("expected OK message, got kind %v", message.Kind)
	}

	if message.Payload != "room=market" {
		t.Errorf("expected payload %q, got %q", "room=market", message.Payload)
	}

	roomID, err := gameWorld.GetPlayerRoom("Alice")
	if err != nil {
		t.Fatalf("get player's room: %v", err)
	}

	if roomID != "market" {
		t.Errorf("expected Alice in market, got %q", roomID)
	}
}

// TestMoveCommandRejectsMissingExit checks the protocol error for invalid movement.
func TestMoveCommandRejectsMissingExit(t *testing.T) {
	gameWorld := createCommandTestWorld(t)
	addTestPlayer(t, gameWorld, "Alice")

	command := commands.NewMoveCommand()

	message, err := command.Execute(&commands.Context{
		PlayerName: "Alice",
		World:      gameWorld,
		RawArgs:    []string{"west"},
	})
	if err != nil {
		t.Fatalf("expected protocol error without Go error, got %v", err)
	}

	if message.Kind != protocol.KindERR {
		t.Fatalf("expected ERR message, got kind %v", message.Kind)
	}

	if message.Code != 301 {
		t.Errorf("expected error code 301, got %d", message.Code)
	}

	if message.Name != "NO_EXIT" {
		t.Errorf("expected error name NO_EXIT, got %q", message.Name)
	}

	roomID, err := gameWorld.GetPlayerRoom("Alice")
	if err != nil {
		t.Fatalf("get player's room: %v", err)
	}

	if roomID != "harbor_square" {
		t.Errorf("player should remain in harbor_square, got %q", roomID)
	}
}

// TestMoveCommandRejectsIncorrectArguments checks argument validation.
func TestMoveCommandRejectsIncorrectArguments(t *testing.T) {
	gameWorld := createCommandTestWorld(t)
	addTestPlayer(t, gameWorld, "Alice")

	tests := []struct {
		name string
		args []string
	}{
		{name: "no direction", args: nil},
		{name: "too many directions", args: []string{"north", "south"}},
		{name: "empty direction", args: []string{""}},
	}

	command := commands.NewMoveCommand()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := command.Execute(&commands.Context{
				PlayerName: "Alice",
				World:      gameWorld,
				RawArgs:    test.args,
			})

			if err == nil {
				t.Fatal("expected an error for invalid arguments")
			}
		})
	}
}

// TestMoveCommandRejectsNilContext checks nil context handling.
func TestMoveCommandRejectsNilContext(t *testing.T) {
	command := commands.NewMoveCommand()

	_, err := command.Execute(nil)
	if err == nil {
		t.Fatal("expected an error when context is nil")
	}
}

// TestQuitCommandReturnsBye checks the disconnect contract.
func TestQuitCommandReturnsBye(t *testing.T) {
	command := commands.NewQuitCommand()

	message, err := command.Execute(nil)

	if !errors.Is(err, commands.ErrQuit) {
		t.Fatalf("expected commands.ErrQuit, got %v", err)
	}

	if message.Kind != protocol.KindOK {
		t.Fatalf("expected OK message, got kind %v", message.Kind)
	}

	if message.Payload != "bye" {
		t.Errorf("expected payload %q, got %q", "bye", message.Payload)
	}
}
