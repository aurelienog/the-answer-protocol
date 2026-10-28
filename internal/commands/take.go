package commands

import (
	"strings"

	"the-answer-protocol/internal/protocol"
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
		return protocol.ERR(500, "INTERNAL_ERROR"), nil
	}

	itemName := strings.TrimSpace(ctx.RawText)
	if itemName == "" {
		return protocol.ERR(400, "ITEM_REQUIRED"), nil
	}

	roomID, err := ctx.World.GetPlayerRoom(ctx.PlayerName)
	if err != nil {
		return protocol.ERR(404, "PLAYER_NOT_FOUND"), nil
	}

	item, err := ctx.World.FindRoomItem(roomID, itemName)
	if err != nil {
		return protocol.ERR(404, "ITEM_NOT_FOUND"), nil
	}

	if err := ctx.World.TakeItem(ctx.PlayerName, item.ID); err != nil {
		return protocol.ERR(409, "TAKE_FAILED"), nil
	}

	return protocol.OK("taken=" + item.ID), nil
}