package commands

import (
	"strconv"
	"strings"

	"the-answer-protocol/internal/protocol"
)

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
		if participant.ID == targetID &&
			!participant.IsPlayer &&
			participant.Alive &&
			!participant.Escaped {
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

	// End combat before publishing death notifications.
	combatEnded := !combatHasLivingPlayers(result) ||
		!combatHasLivingNPCs(result)

	if combatEnded {
		result, err = endCombat(ctx, result.ID)
		if err != nil {
			return mapCombatError(err), nil
		}
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
