package commands

import (
	"strconv"
	"strings"

	"the-answer-protocol/internal/protocol"
)

func (c *CombatCommand) handleHelp(
	ctx *Context,
) (protocol.Message, error) {
	if len(ctx.RawArgs) != 1 {
		return protocol.ERR(
			protocol.CodeInvalidArguments,
			"INVALID_ARGUMENTS",
		), nil
	}

	combat, event, err := ctx.World.RequestCombatHelp(ctx.PlayerName)
	if err != nil {
		return mapCombatError(err), nil
	}

	if event != nil && ctx.Events != nil {
		ctx.Events.Publish(*event)
	}

	return protocol.OK(
		"help_requested=true combat=" + combat.ID,
	), nil
}

func (c *CombatCommand) handleJoin(
	ctx *Context,
) (protocol.Message, error) {
	if len(ctx.RawArgs) != 2 {
		return protocol.ERR(
			protocol.CodeInvalidArguments,
			"INVALID_ARGUMENTS",
		), nil
	}

	combatID := strings.TrimSpace(ctx.RawArgs[1])
	if combatID == "" {
		return protocol.ERR(
			protocol.CodeInvalidArguments,
			"INVALID_ARGUMENTS",
		), nil
	}

	combat, event, err := ctx.World.JoinCombat(
		ctx.PlayerName,
		combatID,
	)
	if err != nil {
		return mapCombatError(err), nil
	}

	if event != nil && ctx.Events != nil {
		ctx.Events.Publish(*event)
	}

	var targetHP int

	for _, participant := range combat.Participants {
		if !participant.IsPlayer {
			targetHP = participant.Stats.HP
			break
		}
	}

	return protocol.OK(
		"joined=true combat=" + combat.ID +
			" target_hp=" + strconv.Itoa(targetHP),
	), nil
}