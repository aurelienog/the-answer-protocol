package commands

import (
	"errors"
	"strings"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

type MoveCommand struct{}

func NewMoveCommand() *MoveCommand {
	return &MoveCommand{}
}

func (c *MoveCommand) Name() string {
	return "MOVE"
}

func (c *MoveCommand) Execute(ctx *Context) (protocol.Message, error) {
	if ctx == nil || ctx.World == nil {
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil
	}

	if strings.TrimSpace(ctx.PlayerName) == "" {
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil
	}

	if len(ctx.RawArgs) != 1 {
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
	}

	direction := strings.ToLower(strings.TrimSpace(ctx.RawArgs[0]))
	if direction == "" {
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
	}

	event, err := ctx.World.MovePlayer(ctx.PlayerName, direction)
	if err != nil {
		switch {
		case errors.Is(err, world.ErrNoExit):
			return protocol.ERR(protocol.CodeNoExit, "NO_EXIT"), nil

		case errors.Is(err, world.ErrPlayerNotFound),
			errors.Is(err, world.ErrInvalidWorldState):
			return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil

		case errors.Is(err, world.ErrInvalidArgument):
			return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil

		default:
			return protocol.ERR(protocol.CodeCommandFailed, "COMMAND_FAILED"), nil
		}
	}

	if ctx.Events != nil {
		ctx.Events.Publish(event)
	}

	return protocol.OK("room=" + event.NewRoomID), nil
}