package commands

import (
	"errors"
    "the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

func mapCombatError(err error) protocol.Message {
	switch {
	case errors.Is(err, world.ErrInvalidArgument):
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS")

	case errors.Is(err, world.ErrPlayerNotInActiveCombat):
		return protocol.ERR(protocol.CodeCommandFailed, "NOT_IN_COMBAT")

	case errors.Is(err, world.ErrInvalidCombatTurn):
		return protocol.ERR(protocol.CodeCommandFailed, "NOT_YOUR_TURN")

	case errors.Is(err, world.ErrInvalidCombatTarget):
		return protocol.ERR(protocol.CodeCommandFailed, "INVALID_TARGET")

	case errors.Is(err, world.ErrNPCNotFound):
		return protocol.ERR(protocol.CodeNPCNotFound, "NPC_NOT_FOUND")

	case errors.Is(err, world.ErrPlayerNotFound),
		errors.Is(err, world.ErrInvalidWorldState):
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR")
	
	case errors.Is(err, world.ErrItemNotInInventory):
		return protocol.ERR(
			protocol.CodeItemNotInInventory,
			"ITEM_NOT_IN_INVENTORY",
		)

	case errors.Is(err, world.ErrItemNotConsumable):
		return protocol.ERR(
			protocol.CodeCommandFailed,
			"ITEM_NOT_CONSUMABLE",
		)

	case errors.Is(err, world.ErrItemHasNoEffect):
		return protocol.ERR(
			protocol.CodeCommandFailed,
			"ITEM_HAS_NO_EFFECT",
		)

	case errors.Is(err, world.ErrPlayerAlreadyInCombat):
		return protocol.ERR(
			protocol.CodeCommandFailed,
			"ALREADY_IN_COMBAT",
		)

	case errors.Is(err, world.ErrItemNotFound):
		return protocol.ERR(
			protocol.CodeItemNotFound,
			"ITEM_NOT_FOUND",
		)

	case errors.Is(err, world.ErrItemNameAmbiguous):
		return protocol.ERR(
			protocol.CodeInvalidArguments,
			"AMBIGUOUS_ITEM_NAME",
		)

	case errors.Is(err, world.ErrCombatHelpNotRequested):
		return protocol.ERR(
			protocol.CodeCombatHelpRequired,
			"COMBAT_HELP_REQUIRED",
		)

	default:
		return protocol.ERR(protocol.CodeCommandFailed, "COMMAND_FAILED")
	}
}