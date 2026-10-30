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

	conversation, err := ctx.World.TalkToNPC(
		ctx.PlayerName,
		target,
	)
	if err != nil {
		return mapTalkError(err)
	}

	return protocol.OKJSON(conversation)
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