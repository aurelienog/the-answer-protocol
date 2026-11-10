package commands

import (
	"encoding/json"
	"errors"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

type InventoryCommand struct{}

func NewInventoryCommand() *InventoryCommand {
	return &InventoryCommand{}
}

func (c *InventoryCommand) Name() string {
	return "INVENTORY"
}

func (c *InventoryCommand) Execute(ctx *Context) (protocol.Message, error) {
	if ctx == nil || ctx.World == nil {
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil
	}

	if len(ctx.RawArgs) != 0 {
		return protocol.ERR(
			protocol.CodeInvalidArguments,
			"INVALID_ARGUMENTS",
		), nil
	}

	items, err := ctx.World.GetInventoryWithQuantities(ctx.PlayerName)
	if err != nil {
		switch {
		case errors.Is(err, world.ErrPlayerNotFound),
			errors.Is(err, world.ErrInvalidWorldState):
			return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil

		default:
			return protocol.ERR(protocol.CodeCommandFailed, "COMMAND_FAILED"), nil
		}
	}

	payload, err := json.Marshal(items)
	if err != nil {
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil
	}

	return protocol.OK(string(payload)), nil
}
