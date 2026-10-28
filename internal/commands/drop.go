package commands

import (
	"strings"

	"the-answer-protocol/internal/protocol"
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
		return protocol.ERR(500, "INTERNAL_ERROR"), nil
	}

	itemName := strings.TrimSpace(ctx.RawText)
	if itemName == "" {
		return protocol.ERR(400, "ITEM_REQUIRED"), nil
	}

	inventory, err := ctx.World.GetInventory(ctx.PlayerName)
	if err != nil {
		return protocol.ERR(404, "PLAYER_NOT_FOUND"), nil
	}

	var itemID string
	for _, item := range inventory {
		if strings.EqualFold(item.ID, itemName) ||
			strings.EqualFold(item.Name, itemName) {
			itemID = item.ID
			break
		}
	}

	if itemID == "" {
		return protocol.ERR(404, "ITEM_NOT_IN_INVENTORY"), nil
	}

	if err := ctx.World.DropItem(ctx.PlayerName, itemID); err != nil {
		return protocol.ERR(409, "DROP_FAILED"), nil
	}

	return protocol.OK("dropped=" + itemID), nil
}
