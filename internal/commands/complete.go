package commands

import (
	"errors"
	"strings"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

type CompleteCommand struct{}

func NewCompleteCommand() *CompleteCommand {
	return &CompleteCommand{}
}

func (c *CompleteCommand) Name() string {
	return "COMPLETE"
}

func (c *CompleteCommand) Execute(ctx *Context) (protocol.Message, error) {
	if ctx == nil || ctx.World == nil {
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil
	}

	questID := strings.TrimSpace(strings.Join(ctx.RawArgs, " "))
	if questID == "" {
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
	}

	err := ctx.World.CompleteQuest(ctx.PlayerName, questID)
	if err != nil {
		switch {
		case errors.Is(err, world.ErrPlayerNotFound),
			errors.Is(err, world.ErrInvalidWorldState):
			return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil

		case errors.Is(err, world.ErrInvalidArgument):
			return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil

		case errors.Is(err, world.ErrQuestNotFound):
			return protocol.ERR(protocol.CodeCommandFailed, "QUEST_NOT_FOUND"), nil

		case errors.Is(err, world.ErrNoQuestAvailable):
			return protocol.ERR(protocol.CodeCommandFailed, "QUEST_NOT_ACTIVE"), nil

		case errors.Is(err, world.ErrItemNotInInventory):
			return protocol.ERR(protocol.CodeCommandFailed, "REQUIRED_ITEM_MISSING"), nil

		case errors.Is(err, world.ErrItemNotFound):
			return protocol.ERR(protocol.CodeCommandFailed, "ITEM_NOT_FOUND"), nil

		case errors.Is(err, world.ErrNPCNotFound):
			return protocol.ERR(protocol.CodeCommandFailed, "QUEST_GIVER_NOT_AVAILABLE"), nil

		default:
			return protocol.ERR(protocol.CodeCommandFailed, "COMMAND_FAILED"), nil
		}
	}

	return protocol.OKJSON(map[string]string{
		"quest_id": questID,
		"status":   string(world.QuestStatusCompleted),
		"message":  "Quest completed",
	})
}