package commands

import (
    "errors"
    "sort"
    "strconv"
    "strings"

    "the-answer-protocol/internal/protocol"
    "the-answer-protocol/internal/world"
)

type CombatCommand struct{}

func NewCombatCommand() *CombatCommand {
	return &CombatCommand{}
}

func (c *CombatCommand) Name() string {
	return "COMBAT"
}

func (c *CombatCommand) Execute(ctx *Context) (protocol.Message, error) {
	if ctx == nil || ctx.World == nil {
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil
	}

	if strings.TrimSpace(ctx.PlayerName) == "" {
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil
	}

	if len(ctx.RawArgs) == 0 {
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
	}

	switch strings.ToUpper(ctx.RawArgs[0]) {
	case "ATTACK":
		return c.handleAttack(ctx)

	case "FLEE":
		return c.handleFlee(ctx)
	
	case "DEFEND":
    	return c.handleDefend(ctx)
	case "USE":
    	return c.handleUse(ctx)

	default:
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
	}
}

func (c *CombatCommand) handleAttack(
	ctx *Context,
) (protocol.Message, error) {
	if len(ctx.RawArgs) < 2 {
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
	}

	targetName := strings.TrimSpace(strings.Join(ctx.RawArgs[1:], " "))
	if targetName == "" {
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
	}

	combat, err := ctx.World.GetActiveCombatForPlayer(ctx.PlayerName)
	if err != nil {
		return mapCombatError(err), nil
	}

	targetID, err := ctx.World.ResolveNPCID(ctx.PlayerName, targetName)
	if err != nil {
		return mapCombatError(err), nil
	}

	isCombatParticipant := false
	for _, participant := range combat.Participants {
		if participant.ID == targetID && !participant.IsPlayer {
			isCombatParticipant = true
			break
		}
	}

	if !isCombatParticipant {
		return protocol.ERR(protocol.CodeCommandFailed, "INVALID_TARGET"), nil
	}

	// Resolve the target within the current combat.
	result, damage, attackEvent, damageEvent, deathEvent, err := ctx.World.AttackCombatParticipant(
		combat.ID,
		ctx.PlayerName,
		targetID,
	)
	if err != nil {
		return mapCombatError(err), nil
	}

	if attackEvent != nil && ctx.Events != nil {
		ctx.Events.Publish(*attackEvent)
	}

	if damageEvent != nil && ctx.Events != nil {
		ctx.Events.Publish(*damageEvent)
	}

	if deathEvent != nil && ctx.Events != nil {
		ctx.Events.Publish(*deathEvent)
	}

	// Process NPC turns automatically.
	if result.Active {
		if err := c.processNPCTurns(ctx, result.ID); err != nil {
			return mapCombatError(err), nil
		}
	}

	// Retrieve the final combat state.
	result, err = ctx.World.GetCombat(result.ID)
	if err != nil {
		return mapCombatError(err), nil
	}

	return protocol.OK(
		"target=" + targetID +
			" damage=" + strconv.Itoa(damage) +
			" combat=" + result.ID,
	), nil
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

	default:
		return protocol.ERR(protocol.CodeCommandFailed, "COMMAND_FAILED")
	}
}

func (c *CombatCommand) processNPCTurns(
	ctx *Context,
	combatID string,
) error {
	for {
		combat, err := ctx.World.GetCombat(combatID)
		if err != nil {
			return err
		}

		// Check whether the combat has ended.
		if !combatHasLivingPlayers(combat) ||
			!combatHasLivingNPCs(combat) {
			_, err = ctx.World.EndCombat(combatID)
			return err
		}

		// Start a new round when all actions are exhausted.
		roundComplete, err := ctx.World.IsCombatRoundComplete(combatID)
		if err != nil {
			return err
		}

		if roundComplete {
			var turnEvent *world.CombatTurnChanged

			combat, turnEvent, err = ctx.World.StartCombatRound(combatID)
			if err != nil {
				return err
			}

			if turnEvent != nil && ctx.Events != nil {
				ctx.Events.Publish(*turnEvent)
			}
		}

		// Find the next participant with available actions.
		var turnEvent *world.CombatTurnChanged

		combat, turnEvent, err = ctx.World.NextCombatTurn(combatID)
		if err != nil {
			return err
		}

		if turnEvent != nil && ctx.Events != nil {
			ctx.Events.Publish(*turnEvent)
		}

		if combat.TurnIndex < 0 ||
			combat.TurnIndex >= len(combat.Participants) {
			return world.ErrInvalidWorldState
		}

		participant := combat.Participants[combat.TurnIndex]

		// Stop when it is a player's turn.
		if participant.IsPlayer {
			return nil
		}
		var (
			attackEvent *world.CombatAttack
			damageEvent *world.CombatDamage
			deathEvent  *world.ParticipantDied
		)
		// Execute the NPC action.
		combat, _, attackEvent, damageEvent, deathEvent, err = ctx.World.ExecuteNPCCombatTurn(combatID)
		if err != nil {
			return err
		}

		if attackEvent != nil && ctx.Events != nil {
			ctx.Events.Publish(*attackEvent)
		}

		if damageEvent != nil && ctx.Events != nil {
			ctx.Events.Publish(*damageEvent)
		}

		if deathEvent != nil && ctx.Events != nil {
			ctx.Events.Publish(*deathEvent)
		}
	}
}

func combatHasLivingPlayers(combat *world.Combat) bool {
	if combat == nil {
		return false
	}

	for _, participant := range combat.Participants {
		if participant.IsPlayer && participant.Alive && !participant.Escaped {
			return true
		}
	}

	return false
}

func combatHasLivingNPCs(combat *world.Combat) bool {
	for _, participant := range combat.Participants {
		if !participant.IsPlayer &&
			participant.Alive &&
			!participant.Escaped {
			return true
		}
	}

	return false
}

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