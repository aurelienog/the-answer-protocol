package commands

import (
	"errors"
	"strings"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

type QuestCommand struct{}

func NewQuestCommand() *QuestCommand {
	return &QuestCommand{}
}

func (c *QuestCommand) Name() string {
	return "QUEST"
}

func (c *QuestCommand) Execute(ctx *Context) (protocol.Message, error) {
	if ctx == nil || ctx.World == nil {
		return protocol.ERR(
			protocol.CodeInternalError,
			"INTERNAL_ERROR",
		), nil
	}

	if len(ctx.RawArgs) > 1 {
		return protocol.ERR(
			protocol.CodeInvalidArguments,
			"INVALID_ARGUMENTS",
		), nil
	}

	if len(ctx.RawArgs) == 1 {
		questID := strings.TrimSpace(ctx.RawArgs[0])
		if questID == "" {
			return protocol.ERR(
				protocol.CodeInvalidArguments,
				"INVALID_ARGUMENTS",
			), nil
		}

		quest, err := ctx.World.GetPlayerQuestSummary(
			ctx.PlayerName,
			questID,
		)
		if err != nil {
			return mapQuestError(err)
		}

		return protocol.OKJSON(quest)
	}

	quests, err := ctx.World.GetPlayerQuests(ctx.PlayerName)
	if err != nil {
		return mapQuestError(err)
	}

	return protocol.OKJSON(quests)
}

func mapQuestError(err error) (protocol.Message, error) {
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

	case errors.Is(err, world.ErrQuestNotFound),
		errors.Is(err, world.ErrNoQuestAvailable):
		return protocol.ERR(
			protocol.CodeCommandFailed,
			"QUEST_NOT_AVAILABLE",
		), nil

	default:
		return protocol.ERR(
			protocol.CodeCommandFailed,
			"COMMAND_FAILED",
		), nil
	}
}
