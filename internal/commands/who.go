package commands

import (
	"errors"
	"strconv"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

type WhoCommand struct{}

func NewWhoCommand() *WhoCommand {
	return &WhoCommand{}
}

func (c *WhoCommand) Name() string {
	return "WHO"
}

func (c *WhoCommand) Execute(ctx *Context) (protocol.Message, error) {
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

	count, err := ctx.World.GetPlayerCount()
	if err != nil {
		return mapWhoError(err)
	}

	return protocol.OK(
		"players=" + strconv.Itoa(count),
	), nil
}

func mapWhoError(err error) (protocol.Message, error) {
	switch {
	case errors.Is(err, world.ErrInvalidWorldState):
		return protocol.ERR(
			protocol.CodeInternalError,
			"INTERNAL_ERROR",
		), nil

	default:
		return protocol.ERR(
			protocol.CodeCommandFailed,
			"COMMAND_FAILED",
		), nil
	}
}