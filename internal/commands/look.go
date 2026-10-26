package commands

import (
	"fmt"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

type lookResponse struct {
	Room    world.Room `json:"room"`
	Players []string   `json:"players"`
	Items   []string   `json:"items"`
	NPCs    []string   `json:"npcs"`
}

type LookCommand struct{}

func NewLookCommand() *LookCommand {
	return &LookCommand{}
}

func (c *LookCommand) Name() string {
	return "look"
}

func (c *LookCommand) Execute(
	ctx *Context,
) (protocol.Message, error) {
	if ctx.World == nil {
		return protocol.Message{}, fmt.Errorf(
			"LOOK: world is not initialized",
		)
	}

	// Get the player's current room from the world.
	roomID, err := ctx.World.GetPlayerRoom(ctx.PlayerName)
	if err != nil {
		return protocol.Message{}, fmt.Errorf("LOOK: %w", err)
	}

	// Get a snapshot of the room.
	room, err := ctx.World.GetRoom(roomID)
	if err != nil {
		return protocol.Message{}, fmt.Errorf("LOOK: %w", err)
	}

	// Get the players currently in the room.
	players, err := ctx.World.GetPlayersInRoom(roomID)
	if err != nil {
		return protocol.Message{}, fmt.Errorf("LOOK: %w", err)
	}

	response := lookResponse{
		Room:    room,
		Players: players,
		Items:   room.Items,
		NPCs:    room.NPCs,
	}

	return protocol.OKJSON(response)
}
