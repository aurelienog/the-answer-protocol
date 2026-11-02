package commands

import (
	"errors"
	"strings"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

type AttackCommand struct{}

func NewAttackCommand() *AttackCommand {
	return &AttackCommand{}
}

func (c *AttackCommand) Name() string {
	return "ATTACK"
}

func (c *AttackCommand) Execute(ctx *Context) (protocol.Message, error) {
	if ctx == nil || ctx.World == nil {
		return protocol.ERR(
			protocol.CodeInternalError,
			"INTERNAL_ERROR",
		), nil
	}

	if len(ctx.RawArgs) == 0 {
		return protocol.ERR(
			protocol.CodeInvalidArguments,
			"INVALID_ARGUMENTS",
		), nil
	}

	target := strings.TrimSpace(strings.Join(ctx.RawArgs, " "))
	if target == "" {
		return protocol.ERR(
			protocol.CodeInvalidArguments,
			"INVALID_ARGUMENTS",
		), nil
	}

	combat, err := ctx.World.StartCombat(ctx.PlayerName, target)
	if err != nil {
		return mapAttackError(err)
	}

	combat, err = ctx.World.InitializeCombatTurns(combat.ID)
	if err != nil {
		_, _ = ctx.World.EndCombat(combat.ID)
		return mapCombatError(err), nil
	}

	var attackerHP int
	var targetHP int

	for _, participant := range combat.Participants {
		if participant.IsPlayer && participant.ID == ctx.PlayerName {
			attackerHP = participant.Stats.HP
		}

		if !participant.IsPlayer {
			targetHP = participant.Stats.HP
		}
	}

	response := struct {
		CombatID   string `json:"combat_id"`
		AttackerHP int    `json:"attacker_hp"`
		TargetHP   int    `json:"target_hp"`
		Damage     int    `json:"damage"`
		Status     string `json:"status"`
	}{
		CombatID:   combat.ID,
		AttackerHP: attackerHP,
		TargetHP:   targetHP,
		Damage:     0,
		Status:     "combat",
	}

	return protocol.OKJSON(response)
}

func mapAttackError(err error) (protocol.Message, error) {
	switch {
	case errors.Is(err, world.ErrPlayerNotFound),
		errors.Is(err, world.ErrInvalidWorldState):
		return protocol.ERR(
			protocol.CodeInternalError,
			"INTERNAL_ERROR",
		), nil

	case errors.Is(err, world.ErrInvalidArgument):
		return protocol.ERR(
			protocol.CodeInvalidArguments,
			"INVALID_ARGUMENTS",
		), nil

	case errors.Is(err, world.ErrNPCNotFound):
		return protocol.ERR(
			protocol.CodeNPCNotFound,
			"NPC_NOT_FOUND",
		), nil

	case errors.Is(err, world.ErrNPCNotHostile):
		return protocol.ERR(
			protocol.CodeNPCNotHostile,
			"NPC_NOT_HOSTILE",
		), nil

	case errors.Is(err, world.ErrPlayerAlreadyInCombat):
		return protocol.ERR(
			protocol.CodeCommandFailed,
			"PLAYER_ALREADY_IN_COMBAT",
		), nil

	case errors.Is(err, world.ErrNPCAlreadyInCombat):
		return protocol.ERR(
			protocol.CodeCommandFailed,
			"NPC_ALREADY_IN_COMBAT",
		), nil

	default:
		return protocol.ERR(
			protocol.CodeCommandFailed,
			"COMMAND_FAILED",
		), nil
	}
}