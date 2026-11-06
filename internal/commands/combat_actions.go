package commands

import (
    "sort"
    "strconv"
    "strings"

    "the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

func (c *CombatCommand) handleDefend(
    ctx *Context,
) (protocol.Message, error) {
    if len(ctx.RawArgs) != 1 {
        return protocol.ERR(
            protocol.CodeInvalidArguments,
            "INVALID_ARGUMENTS",
        ), nil
    }

    combat, err := ctx.World.GetActiveCombatForPlayer(ctx.PlayerName)
    if err != nil {
        return mapCombatError(err), nil
    }

    result, turnEvent, err := ctx.World.DefendCombatParticipant(
        combat.ID,
        ctx.PlayerName,
    )
    if err != nil {
        return mapCombatError(err), nil
    }

    if turnEvent != nil && ctx.Events != nil {
        ctx.Events.Publish(*turnEvent)
    }

    // Process NPC turns automatically.
    if result.Active {
        if err := c.processNPCTurns(ctx, result.ID); err != nil {
            return mapCombatError(err), nil
        }
    }

    result, err = ctx.World.GetCombat(result.ID)
    if err != nil {
        return mapCombatError(err), nil
    }

    return protocol.OK(
		"defended=true" +
			" defense_bonus=" + strconv.Itoa(world.DefaultDefenseBonusPercent) +
			" combat=" + result.ID,
	), nil
}

func (c *CombatCommand) handleUse(
	ctx *Context,
) (protocol.Message, error) {
	if len(ctx.RawArgs) < 2 {
		return protocol.ERR(
			protocol.CodeInvalidArguments,
			"INVALID_ARGUMENTS",
		), nil
	}

	itemName := strings.TrimSpace(
		strings.Join(ctx.RawArgs[1:], " "),
	)

	if itemName == "" {
		return protocol.ERR(
			protocol.CodeInvalidArguments,
			"ITEM_REQUIRED",
		), nil
	}

	// Retrieve the player's active combat.
	combat, err := ctx.World.GetActiveCombatForPlayer(
		ctx.PlayerName,
	)
	if err != nil {
		return mapCombatError(err), nil
	}

	// Retrieve the player's inventory.
	inventory, err := ctx.World.GetInventory(ctx.PlayerName)
	if err != nil {
		return mapCombatError(err), nil
	}

	// Resolve the item by ID or name.
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

	// Consume the item during combat.
	result, event, err := ctx.World.CombatUseItem(
		combat.ID,
		ctx.PlayerName,
		itemID,
	)
	if err != nil {
		return mapCombatError(err), nil
	}

	// Publish the item usage event.
	if ctx.Events != nil {
		ctx.Events.Publish(event)
	}

	// Build a deterministic response.
	stats := make([]string, 0, len(event.StatChanges))

	for stat := range event.StatChanges {
		stats = append(stats, stat)
	}

	sort.Strings(stats)

	var changes []string

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

	response += " combat=" + result.ID

	return protocol.OK(response), nil
}

func (c *CombatCommand) handleFlee(
    ctx *Context,
) (protocol.Message, error) {
    if len(ctx.RawArgs) != 1 {
        return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
    }

    combat, err := ctx.World.GetActiveCombatForPlayer(ctx.PlayerName)
    if err != nil {
        return mapCombatError(err), nil
    }

    result, fleeEvent, err := ctx.World.FleeCombat(
        combat.ID,
        ctx.PlayerName,
    )
    if err != nil {
        return mapCombatError(err), nil
    }

    if fleeEvent != nil && ctx.Events != nil {
        ctx.Events.Publish(*fleeEvent)
    }

    // Check whether the combat should end.
    if !combatHasLivingPlayers(result) ||
        !combatHasLivingNPCs(result) {
        _, err = ctx.World.EndCombat(result.ID)
        if err != nil {
            return mapCombatError(err), nil
        }
    } else if result.Active {
        // Process NPC turns automatically.
        if err := c.processNPCTurns(ctx, result.ID); err != nil {
            return mapCombatError(err), nil
        }
    }

    result, err = ctx.World.GetCombat(result.ID)
    if err != nil {
        return mapCombatError(err), nil
    }

    response := "fled=true combat=" + result.ID

	if fleeEvent != nil {
		response = "fled=true" +
			" damage=" + strconv.Itoa(fleeEvent.Damage) +
			" hp=" + strconv.Itoa(fleeEvent.RemainingHP) +
			" combat=" + result.ID
	}

	return protocol.OK(response), nil
}
