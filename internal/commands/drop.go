package commands

import (
	"errors"
	"strings"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

type DropCommand struct{}

func NewDropCommand() *DropCommand {
	return &DropCommand{}
}

func (c *DropCommand) Name() string {
	return "DROP"
}

func (c *DropCommand) Execute(ctx *Context) (protocol.Message, error) {
	if ctx == nil || ctx.World == nil {
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil
	}

	itemName := strings.TrimSpace(ctx.RawText)
	if itemName == "" {
		return protocol.ERR(protocol.CodeInvalidRequest, "ITEM_REQUIRED"), nil
	}

	inventory, err := ctx.World.GetInventory(ctx.PlayerName)
	if err != nil {
		switch {
		case errors.Is(err, world.ErrPlayerNotFound),
			errors.Is(err, world.ErrInvalidWorldState):
			return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil

		default:
			return protocol.ERR(protocol.CodeCommandFailed, "COMMAND_FAILED"), nil
		}
	}

	var itemID string

	for _, item := range inventory {
		if strings.EqualFold(item.ID, itemName) ||
			strings.EqualFold(item.Name, itemName) {
			if itemID != "" && itemID != item.ID {
				return protocol.ERR(protocol.CodeInvalidArguments, "AMBIGUOUS_ITEM_NAME"), nil
			}

			itemID = item.ID
		}
	}

	if itemID == "" {
		return protocol.ERR(protocol.CodeItemNotInInventory, "ITEM_NOT_IN_INVENTORY"), nil
	}

	event, err := ctx.World.DropItem(ctx.PlayerName, itemID)
	if err != nil {
		switch {
		case errors.Is(err, world.ErrPlayerNotFound),
			errors.Is(err, world.ErrInvalidWorldState):
			return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil

		case errors.Is(err, world.ErrItemNotFound):
			return protocol.ERR(protocol.CodeItemNotFound, "ITEM_NOT_FOUND"), nil

		case errors.Is(err, world.ErrItemNotInInventory):
			return protocol.ERR(protocol.CodeItemNotInInventory, "ITEM_NOT_IN_INVENTORY"), nil

		case errors.Is(err, world.ErrItemCannotBeDropped):
			return protocol.ERR(protocol.CodeItemCannotBeDropped, "ITEM_CANNOT_BE_DROPPED"), nil

		default:
			return protocol.ERR(protocol.CodeCommandFailed, "COMMAND_FAILED"), nil
		}
	}

	if ctx.Events != nil {
		ctx.Events.Publish(event)
	}

	return protocol.OK("dropped=" + itemID), nil
}
