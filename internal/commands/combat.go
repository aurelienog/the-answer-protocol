package commands

import (
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

	case "HELP":
		return c.handleHelp(ctx)

	case "JOIN":
		return c.handleJoin(ctx)

	default:
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
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
		} else {
			// Advance to the next participant in the current round.
			var turnEvent *world.CombatTurnChanged

			combat, turnEvent, err = ctx.World.NextCombatTurn(combatID)
			if err != nil {
				return err
			}

			if turnEvent != nil && ctx.Events != nil {
				ctx.Events.Publish(*turnEvent)
			}
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
		combat, _, attackEvent, damageEvent, deathEvent, err =
			ctx.World.ExecuteNPCCombatTurn(combatID)
		if err != nil {
			return err
		}

		// End combat before publishing death notifications.
		combatEnded := !combatHasLivingPlayers(combat) ||
			!combatHasLivingNPCs(combat)

		if combatEnded {
			combat, err = ctx.World.EndCombat(combatID)
			if err != nil {
				return err
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

		if combatEnded {
			return nil
		}
	}
}

func combatHasLivingPlayers(combat *world.Combat) bool {
	if combat == nil {
		return false
	}

	for _, participant := range combat.Participants {
		if participant.IsPlayer &&
			participant.Alive &&
			participant.Connected &&
			!participant.Escaped {
			return true
		}
	}

	return false
}

func combatHasLivingNPCs(combat *world.Combat) bool {
	if combat == nil {
		return false
	}

	for _, participant := range combat.Participants {
		if !participant.IsPlayer &&
			participant.Alive &&
			!participant.Escaped {
			return true
		}
	}

	return false
}