package commands

import (
	"errors"
	"strings"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

type TakeCommand struct{}

func NewTakeCommand() *TakeCommand {
	return &TakeCommand{}
}

func (c *TakeCommand) Name() string {
	return "TAKE"
}

func (c *TakeCommand) Execute(ctx *Context) (protocol.Message, error) {
	if ctx == nil || ctx.World == nil {
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil
	}

	itemName := strings.TrimSpace(ctx.RawText)
	if itemName == "" {
		return protocol.ERR(protocol.CodeInvalidRequest, "ITEM_REQUIRED"), nil
	}

	roomID, err := ctx.World.GetPlayerRoom(ctx.PlayerName)
	if err != nil {
		switch {
		case errors.Is(err, world.ErrPlayerNotFound),
			errors.Is(err, world.ErrInvalidWorldState):
			return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil

		default:
			return protocol.ERR(protocol.CodeCommandFailed, "COMMAND_FAILED"), nil
		}
	}

	item, err := ctx.World.FindRoomItem(roomID, itemName)
	if err != nil {
		switch {
		case errors.Is(err, world.ErrItemNotFound):
			return protocol.ERR(protocol.CodeItemNotFound, "ITEM_NOT_FOUND"), nil

		case errors.Is(err, world.ErrItemNameAmbiguous):
			return protocol.ERR(protocol.CodeInvalidArguments, "AMBIGUOUS_ITEM_NAME"), nil

		case errors.Is(err, world.ErrInvalidArgument):
			return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil

		case errors.Is(err, world.ErrInvalidWorldState):
			return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil

		default:
			return protocol.ERR(protocol.CodeCommandFailed, "COMMAND_FAILED"), nil
		}
	}

	if err := ctx.World.TakeItem(ctx.PlayerName, item.ID); err != nil {
		switch {
		case errors.Is(err, world.ErrPlayerNotFound),
			errors.Is(err, world.ErrInvalidWorldState):
			return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil

		case errors.Is(err, world.ErrItemNotFound):
			return protocol.ERR(protocol.CodeItemNotFound, "ITEM_NOT_FOUND"), nil

		case errors.Is(err, world.ErrItemNotInRoom):
			return protocol.ERR(protocol.CodeItemNotInRoom, "ITEM_NOT_IN_ROOM"), nil

		case errors.Is(err, world.ErrItemAlreadyOwned):
			return protocol.ERR(protocol.CodeItemAlreadyOwned, "ITEM_ALREADY_OWNED"), nil

		default:
			return protocol.ERR(protocol.CodeCommandFailed, "COMMAND_FAILED"), nil
		}
	}

	return protocol.OK("taken=" + item.ID), nil
}