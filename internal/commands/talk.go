package commands

import (
	"errors"
	"strings"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

type TalkCommand struct{}

func NewTalkCommand() *TalkCommand {
	return &TalkCommand{}
}

func (c *TalkCommand) Name() string {
	return "TALK"
}

func (c *TalkCommand) Execute(ctx *Context) (protocol.Message, error) {
	if ctx == nil || ctx.World == nil {
		return protocol.ERR(
			protocol.CodeInternalError,
			"INTERNAL_ERROR",
		), nil
	}

	if len(ctx.RawArgs) == 0 {
		return protocol.ERR(
			protocol.CodeInvalidArguments,
			"INVALID_ARGUMENTS",
		), nil
	}

	target := strings.TrimSpace(strings.Join(ctx.RawArgs, " "))
	if target == "" {
		return protocol.ERR(
			protocol.CodeInvalidArguments,
			"INVALID_ARGUMENTS",
		), nil
	}

	npcs, err := getNPCsForPlayer(ctx.World, ctx.PlayerName)
	if err != nil {
		return mapTalkError(err)
	}

	npcID, err := resolveNPCID(target, npcs)
	if err != nil {
		return mapTalkError(err)
	}

	conversation, err := ctx.World.TalkToNPC(ctx.PlayerName, npcID)
	if err != nil {
		return mapTalkError(err)
	}

	return protocol.OKJSON(conversation)
}

func getNPCsForPlayer(w *world.World, playerName string) ([]world.NPC, error) {
	roomID, err := w.GetPlayerRoom(playerName)
	if err != nil {
		return nil, err
	}

	return w.GetNPCsInRoom(roomID)
}

func resolveNPCID(target string, npcs []world.NPC) (string, error) {
	normalizedTarget := normalizeNPCTarget(target)

	var matchedID string

	for _, npc := range npcs {
		if normalizeNPCTarget(npc.ID) == normalizedTarget ||
			normalizeNPCTarget(npc.Name) == normalizedTarget {
			if matchedID != "" && matchedID != npc.ID {
				return "", world.ErrInvalidArgument
			}

			matchedID = npc.ID
		}
	}

	if matchedID == "" {
		return "", world.ErrNPCNotFound
	}

	return matchedID, nil
}

func normalizeNPCTarget(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func mapTalkError(err error) (protocol.Message, error) {
	switch {
	case errors.Is(err, world.ErrPlayerNotFound),
		errors.Is(err, world.ErrInvalidWorldState):
		return protocol.ERR(
			protocol.CodeInternalError,
			"INTERNAL_ERROR",
		), nil

	case errors.Is(err, world.ErrInvalidArgument):
		return protocol.ERR(
			protocol.CodeInvalidArguments,
			"INVALID_ARGUMENTS",
		), nil

	case errors.Is(err, world.ErrNPCNotFound):
		return protocol.ERR(
			protocol.CodeCommandFailed,
			"NPC_NOT_AVAILABLE",
		), nil

	default:
		return protocol.ERR(
			protocol.CodeCommandFailed,
			"COMMAND_FAILED",
		), nil
	}
}