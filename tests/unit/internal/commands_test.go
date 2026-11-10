package internal_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/events"
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
				"items": {
					"map": 1
				},
				"npcs": ["guard"]
			},
			"market": {
				"id": "market",
				"name": "Market",
				"description": "A busy market.",
				"exits": {
					"south": "harbor_square"
				},
				"items": {},
				"npcs": []
			}
		},
		"items": {
			"map": {
				"id": "map",
				"name": "Map",
				"description": "A map of the surrounding area.",
				"type": "common"
			}
		},
		"npcs": {
			"guard": {
				"id": "guard",
				"name": "Harbor Guard",
				"combat_stats": {
					"hp": 1,
					"max_hp": 1,
					"attack": 1,
					"defense": 0,
					"speed": 1
				}
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

func assertProtocolError(
	t *testing.T,
	message protocol.Message,
	err error,
	code int,
	name string,
) {
	t.Helper()

	if err != nil {
		t.Fatalf("expected protocol error without Go error, got %v", err)
	}

	if message.Kind != protocol.KindERR {
		t.Fatalf("expected ERR message, got kind %v", message.Kind)
	}

	if message.Code != code {
		t.Errorf("expected error code %d, got %d", code, message.Code)
	}

	if message.Name != name {
		t.Errorf("expected error name %q, got %q", name, message.Name)
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
		Players []string     `json:"players"`
		Items   []world.Item `json:"items"`
		NPCs    []string     `json:"npcs"`
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

	if len(response.Items) != 1 {
		t.Fatalf("expected 1 item in room items, got %d", len(response.Items))
	}

	item := response.Items[0]

	if item.ID != "map" {
		t.Errorf("expected item ID %q, got %q", "map", item.ID)
	}

	if item.Name != "Map" {
		t.Errorf("expected item name %q, got %q", "Map", item.Name)
	}

	if item.Description != "A map of the surrounding area." {
		t.Errorf("unexpected item description: %q", item.Description)
	}

	if item.Type != world.ItemTypeCommon {
		t.Errorf("expected item type %q, got %q", world.ItemTypeCommon, item.Type)
	}

	if len(response.NPCs) != 1 || response.NPCs[0] != "Harbor Guard" {
		t.Errorf("expected Harbor Guard in room NPCs, got %v", response.NPCs)
	}

}

// TestLookCommandReturnsProtocolErrorForNilWorld checks missing world configuration.
func TestLookCommandReturnsProtocolErrorForNilWorld(t *testing.T) {
	command := commands.NewLookCommand()

	message, err := command.Execute(&commands.Context{
		PlayerName: "Alice",
		World:      nil,
	})

	assertProtocolError(
		t,
		message,
		err,
		protocol.CodeInternalError,
		"INTERNAL_ERROR",
	)
}

func TestCommandsWithoutArgumentsRejectUnexpectedArguments(t *testing.T) {
	gameWorld := createCommandTestWorld(t)
	addTestPlayer(t, gameWorld, "Alice")

	dispatcher := commands.NewDispatcher()
	commands.RegisterCommands(dispatcher)

	for _, command := range []string{"LOOK", "STATUS", "INVENTORY", "QUIT"} {
		t.Run(command, func(t *testing.T) {
			message, err := dispatcher.Dispatch(
				&commands.Context{PlayerName: "Alice", World: gameWorld},
				command+" unexpected",
			)

			assertProtocolError(
				t,
				message,
				err,
				protocol.CodeInvalidArguments,
				"INVALID_ARGUMENTS",
			)
		})
	}
}

// TestMoveCommandChangesRoom checks successful movement.
func TestMoveCommandChangesRoom(t *testing.T) {
	gameWorld := createCommandTestWorld(t)
	addTestPlayer(t, gameWorld, "Alice")

	eventBus := events.NewBus()
	var publishedEvents []any
	eventBus.Subscribe(func(event any) {
		publishedEvents = append(publishedEvents, event)
	})

	command := commands.NewMoveCommand()

	message, err := command.Execute(&commands.Context{
		PlayerName: "Alice",
		World:      gameWorld,
		RawArgs:    []string{"north"},
		Events:     eventBus,
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

	if len(publishedEvents) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(publishedEvents))
	}

	moved, ok := publishedEvents[0].(world.PlayerMoved)
	if !ok {
		t.Fatalf("expected world.PlayerMoved, got %T", publishedEvents[0])
	}

	if moved.PlayerName != "Alice" {
		t.Errorf("expected moving player Alice, got %q", moved.PlayerName)
	}

	if moved.PreviousRoomID != "harbor_square" {
		t.Errorf("expected previous room harbor_square, got %q", moved.PreviousRoomID)
	}

	if moved.NewRoomID != "market" {
		t.Errorf("expected new room market, got %q", moved.NewRoomID)
	}

	if len(moved.PreviousRoomPlayers) != 1 || moved.PreviousRoomPlayers[0] != "Alice" {
		t.Errorf("expected Alice in previous room snapshot, got %v", moved.PreviousRoomPlayers)
	}

	if len(moved.NewRoomPlayers) != 0 {
		t.Errorf("expected empty destination room snapshot, got %v", moved.NewRoomPlayers)
	}
}

// TestMoveCommandRejectsMissingExit checks the protocol error for invalid movement.
func TestMoveCommandRejectsMissingExit(t *testing.T) {
	gameWorld := createCommandTestWorld(t)
	addTestPlayer(t, gameWorld, "Alice")

	command := commands.NewMoveCommand()

	eventBus := events.NewBus()
	publishedEvents := 0

	eventBus.Subscribe(func(event any) {
		publishedEvents++
	})

	message, err := command.Execute(&commands.Context{
		PlayerName: "Alice",
		World:      gameWorld,
		RawArgs:    []string{"west"},
		Events:     eventBus,
	})
	assertProtocolError(t, message, err, protocol.CodeNoExit, "NO_EXIT")

	roomID, err := gameWorld.GetPlayerRoom("Alice")
	if err != nil {
		t.Fatalf("get player's room: %v", err)
	}

	if roomID != "harbor_square" {
		t.Errorf("player should remain in harbor_square, got %q", roomID)
	}

	if publishedEvents != 0 {
		t.Errorf("expected no events for failed movement, got %d", publishedEvents)
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
			message, err := command.Execute(&commands.Context{
				PlayerName: "Alice",
				World:      gameWorld,
				RawArgs:    test.args,
			})

			assertProtocolError(
				t,
				message,
				err,
				protocol.CodeInvalidArguments,
				"INVALID_ARGUMENTS",
			)
		})
	}
}

// TestMoveCommandReturnsProtocolErrorForNilContext checks nil context handling.
func TestMoveCommandReturnsProtocolErrorForNilContext(t *testing.T) {
	command := commands.NewMoveCommand()

	message, err := command.Execute(nil)
	assertProtocolError(
		t,
		message,
		err,
		protocol.CodeInternalError,
		"INTERNAL_ERROR",
	)
}

// TestQuitCommandReturnsBye checks the disconnect contract.
func TestQuitCommandReturnsBye(t *testing.T) {
	command := commands.NewQuitCommand()

	message, err := command.Execute(&commands.Context{})

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

func TestQuitCommandReturnsProtocolErrorForNilContext(t *testing.T) {
	command := commands.NewQuitCommand()

	message, err := command.Execute(nil)
	assertProtocolError(
		t,
		message,
		err,
		protocol.CodeInternalError,
		"INTERNAL_ERROR",
	)
}
