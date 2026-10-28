package commands

import (
	"errors"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

type lookResponse struct {
	Room    world.Room   `json:"room"`
	Players []string     `json:"players"`
	Items   []world.Item `json:"items"`
	NPCs    []string     `json:"npcs"`
}

type LookCommand struct{}

func NewLookCommand() *LookCommand {
	return &LookCommand{}
}

func (c *LookCommand) Name() string {
	return "LOOK"
}

func (c *LookCommand) Execute(ctx *Context) (protocol.Message, error) {
	if ctx == nil || ctx.World == nil {
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil
	}

	roomID, err := ctx.World.GetPlayerRoom(ctx.PlayerName)
	if err != nil {
		return mapLookError(err)
	}

	room, err := ctx.World.GetRoom(roomID)
	if err != nil {
		return mapLookError(err)
	}

	players, err := ctx.World.GetPlayersInRoom(roomID)
	if err != nil {
		return mapLookError(err)
	}

	items, err := ctx.World.GetRoomItems(roomID)
	if err != nil {
		return mapLookError(err)
	}

	response := lookResponse{
		Room:    room,
		Players: players,
		Items:   items,
		NPCs:    room.NPCs,
	}

	return protocol.OKJSON(response)
}

func mapLookError(err error) (protocol.Message, error) {
	switch {
	case errors.Is(err, world.ErrPlayerNotFound),
		errors.Is(err, world.ErrInvalidWorldState):
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil

	case errors.Is(err, world.ErrInvalidArgument):
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil

	default:
		return protocol.ERR(protocol.CodeCommandFailed, "COMMAND_FAILED"), nil
	}
}