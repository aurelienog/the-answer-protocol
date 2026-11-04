package commands

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

type UseCommand struct{}

func NewUseCommand() *UseCommand {
	return &UseCommand{}
}

func (c *UseCommand) Name() string {
	return "USE"
}

func (c *UseCommand) Execute(ctx *Context) (protocol.Message, error) {
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
				return protocol.ERR(
					protocol.CodeInvalidArguments,
					"AMBIGUOUS_ITEM_NAME",
				), nil
			}

			itemID = item.ID
		}
	}

	if itemID == "" {
		return protocol.ERR(
			protocol.CodeItemNotInInventory,
			"ITEM_NOT_IN_INVENTORY",
		), nil
	}

	event, err := ctx.World.UseItem(ctx.PlayerName, itemID)
	if err != nil {
		switch {
		case errors.Is(err, world.ErrPlayerNotFound),
			errors.Is(err, world.ErrInvalidWorldState):
			return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil

		case errors.Is(err, world.ErrItemNotInInventory):
			return protocol.ERR(
				protocol.CodeItemNotInInventory,
				"ITEM_NOT_IN_INVENTORY",
			), nil

		case errors.Is(err, world.ErrItemNotConsumable):
			return protocol.ERR(
				protocol.CodeCommandFailed,
				"ITEM_NOT_CONSUMABLE",
			), nil

		case errors.Is(err, world.ErrItemHasNoEffect):
			return protocol.ERR(
				protocol.CodeCommandFailed,
				"ITEM_HAS_NO_EFFECT",
			), nil

		case errors.Is(err, world.ErrPlayerAlreadyInCombat):
			return protocol.ERR(
				protocol.CodeCommandFailed,
				"PLAYER_ALREADY_IN_COMBAT",
			), nil

		default:
			return protocol.ERR(
				protocol.CodeCommandFailed,
				"COMMAND_FAILED",
			), nil
		}
	}

	if ctx.Events != nil {
		ctx.Events.Publish(event)
	}

	changes := make([]string, 0, len(event.StatChanges))

	stats := make([]string, 0, len(event.StatChanges))

	for stat := range event.StatChanges {
		stats = append(stats, stat)
	}

	sort.Strings(stats)

	for _, stat := range stats {
		changes = append(
			changes,
			stat+"="+strconv.Itoa(event.StatChanges[stat]),
		)
	}

	response := "used=" + event.ItemID

	if len(changes) > 0 {
		response += " changes=" + strings.Join(changes, ",")
	}

	return protocol.OK(response), nil
}
