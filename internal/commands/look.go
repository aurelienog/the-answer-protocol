package commands

import (
	"errors"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

type lookResponse struct {
    Room    world.Room         `json:"room"`
    Players []string           `json:"players"`
    Items   []world.Item       `json:"items"`
    NPCs    []world.NPCSummary `json:"npcs"`
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

	if len(ctx.RawArgs) != 0 {
		return protocol.ERR(
			protocol.CodeInvalidArguments,
			"INVALID_ARGUMENTS",
		), nil
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

	npcs, err := ctx.World.GetNPCsInRoom(roomID)
	if err != nil {
		return mapLookError(err)
	}

	npcSummaries := make([]world.NPCSummary, 0, len(npcs))

	for _, npc := range npcs {
		npcSummaries = append(npcSummaries, world.NPCSummary{
			ID:          npc.ID,
			Name:        npc.Name,
			Role:        npc.Role,
			Hostile:     npc.Hostile,
			CombatStats: npc.CombatStats,
		})
	}

	response := lookResponse{
		Room:    room,
		Players: players,
		Items:   items,
		NPCs:    npcSummaries,
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
