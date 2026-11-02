package commands

import (
	"errors"
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

	case errors.Is(err, world.ErrPlayerNotFound),
		errors.Is(err, world.ErrNPCNotFound),
		errors.Is(err, world.ErrInvalidWorldState):
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR")

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
			combat, err = ctx.World.StartCombatRound(combatID)
			if err != nil {
				return err
			}
		}

		// Find the next participant with available actions.
		combat, err = ctx.World.NextCombatTurn(combatID)
		if err != nil {
			return err
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
	for _, participant := range combat.Participants {
		if participant.IsPlayer &&
			participant.Alive &&
			!participant.Escaped {
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