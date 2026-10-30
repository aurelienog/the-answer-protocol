package commands

import (
	"errors"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

type QuestsCommand struct{}

func NewQuestsCommand() *QuestsCommand {
	return &QuestsCommand{}
}

func (c *QuestsCommand) Name() string {
	return "QUESTS"
}

func (c *QuestsCommand) Execute(ctx *Context) (protocol.Message, error) {
	if ctx == nil || ctx.World == nil {
		return protocol.ERR(
			protocol.CodeInternalError,
			"INTERNAL_ERROR",
		), nil
	}

	if len(ctx.RawArgs) != 0 {
		return protocol.ERR(
			protocol.CodeInvalidArguments,
			"INVALID_ARGUMENTS",
		), nil
	}

	quests, err := ctx.World.GetPlayerQuests(ctx.PlayerName)
	if err != nil {
		return mapQuestsError(err)
	}

	return protocol.OKJSON(quests)
}

func mapQuestsError(err error) (protocol.Message, error) {
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

	default:
		return protocol.ERR(
			protocol.CodeCommandFailed,
			"COMMAND_FAILED",
		), nil
	}
}