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

	if len(ctx.RawArgs) == 0 {
		return protocol.ERR(
			protocol.CodeInvalidArguments,
			"INVALID_ARGUMENTS",
		), nil
	}

	npcIdentifier := strings.TrimSpace(strings.Join(ctx.RawArgs, " "))
	if npcIdentifier == "" {
		return protocol.ERR(
			protocol.CodeInvalidArguments,
			"INVALID_ARGUMENTS",
		), nil
	}

	conversation, err := ctx.World.TalkToNPC(
		ctx.PlayerName,
		npcIdentifier,
	)
	if err != nil {
		return mapQuestError(err)
	}

	// The NPC does not offer any quest.
	if conversation.Quest == nil {
		return protocol.ERR(
			protocol.CodeNoQuestAvailable,
			"NO_QUEST_AVAILABLE",
		), nil
	}

	switch conversation.QuestStatus {
	case world.QuestStatusAvailable:
		// QUEST also accepts the quest. ACCEPT is not a TAP command.
		err := ctx.World.AcceptQuest(
			ctx.PlayerName,
			conversation.Quest.ID,
		)
		if err != nil {
			return mapQuestError(err)
		}

	case world.QuestStatusActive,
		world.QuestStatusCompleted,
		world.QuestStatusLocked:
		return protocol.ERR(
			protocol.CodeNoQuestAvailable,
			"NO_QUEST_AVAILABLE",
		), nil

	default:
		return protocol.ERR(
			protocol.CodeNoQuestAvailable,
			"NO_QUEST_AVAILABLE",
		), nil
	}

	return protocol.OKJSON(conversation.Quest)
}

func mapQuestError(err error) (protocol.Message, error) {
	switch {
	case errors.Is(err, world.ErrNPCNotFound):
		return protocol.ERR(
			protocol.CodeNPCNotFound,
			"NPC_NOT_FOUND",
		), nil

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
			protocol.CodeNoQuestAvailable,
			"NO_QUEST_AVAILABLE",
		), nil

	default:
		return protocol.ERR(
			protocol.CodeCommandFailed,
			"COMMAND_FAILED",
		), nil
	}
}