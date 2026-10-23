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

	room, exists := ctx.World.Rooms[ctx.CurrentRoom]
	if !exists {
		return protocol.Message{}, fmt.Errorf(
			"LOOK: room %q does not exist",
			ctx.CurrentRoom,
		)
	}

	response := lookResponse{
		Room:    room,
		Players: []string{},
		Items:   room.Items,
		NPCs:    room.NPCs,
	}

	return protocol.OKJSON(response)
}