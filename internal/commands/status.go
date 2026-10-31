package commands

import (
	"errors"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

type statusResponse struct {
	HP     int    `json:"hp"`
	MaxHP  int    `json:"max_hp"`
	Status string `json:"status"`
}

type StatusCommand struct{}

func NewStatusCommand() *StatusCommand {
	return &StatusCommand{}
}

func (c *StatusCommand) Name() string {
	return "STATUS"
}

func (c *StatusCommand) Execute(ctx *Context) (protocol.Message, error) {
	if ctx == nil || ctx.World == nil {
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil
	}

	stats, err := ctx.World.GetPlayerCombatStats(ctx.PlayerName)
	if err != nil {
		return mapStatusError(err)
	}

	response := statusResponse{
		HP:     stats.HP,
		MaxHP:  stats.MaxHP,
		Status: getPlayerStatus(stats.HP, stats.MaxHP),
	}

	return protocol.OKJSON(response)
}

func getPlayerStatus(hp, maxHP int) string {
	if hp <= 0 {
		return "dead"
	}

	if maxHP <= 0 {
		return "unknown"
	}

	healthPercentage := float64(hp) / float64(maxHP)

	switch {
	case healthPercentage <= 0.25:
		return "critical"
	case healthPercentage <= 0.50:
		return "wounded"
	case healthPercentage <= 0.75:
		return "injured"
	default:
		return "healthy"
	}
}

func mapStatusError(err error) (protocol.Message, error) {
	switch {
	case errors.Is(err, world.ErrPlayerNotFound),
		errors.Is(err, world.ErrInvalidWorldState):
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil

	case errors.Is(err, world.ErrInvalidArgument):
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil

	default:
		return protocol.ERR(protocol.CodeCommandFailed, "COMMAND_FAILED"), nil
	}
}