package commands

import (
	"errors"
	"log"
	"sort"
	"strconv"
	"strings"

	"the-answer-protocol/internal/events"
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

func publishDomainEvents(publisher events.Publisher, domainEvents []any) {
	if publisher == nil {
		return
	}

	for _, event := range domainEvents {
		if event != nil {
			publisher.Publish(event)
		}
	}
}

// NewCombatTurnTimeoutHandler expires an inactive player's turn and publishes
// all resulting combat progression events.
func NewCombatTurnTimeoutHandler(
	gameWorld *world.World,
	publisher events.Publisher,
) func(world.CombatTurnChanged) {
	return func(turn world.CombatTurnChanged) {
		progressionEvents, err := gameWorld.SkipPlayerCombatTurn(
			turn.CombatID,
			turn.ParticipantID,
		)
		if err != nil {
			if !errors.Is(err, world.ErrInvalidCombatTurn) {
				log.Printf(
					"Failed to expire combat turn for participant %q in combat %q: %v",
					turn.ParticipantID,
					turn.CombatID,
					err,
				)
			}
			return
		}

		publishDomainEvents(publisher, progressionEvents)
	}
}

func (c *CombatCommand) Execute(ctx *Context) (protocol.Message, error) {
	if ctx == nil || ctx.World == nil || strings.TrimSpace(ctx.PlayerName) == "" {
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil
	}

	if len(ctx.RawArgs) == 0 {
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
	}

	switch strings.ToUpper(ctx.RawArgs[0]) {
	case "ATTACK":
		return c.handleAttack(ctx)
	case "DEFEND":
		return c.handleDefend(ctx)
	case "USE":
		return c.handleUse(ctx)
	case "FLEE":
		return c.handleFlee(ctx)
	case "HELP":
		return c.handleHelp(ctx)
	case "JOIN":
		return c.handleJoin(ctx)
	default:
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
	}
}

func (c *CombatCommand) handleAttack(ctx *Context) (protocol.Message, error) {
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

	result, damage, domainEvents, err := ctx.World.AttackCombatParticipant(
		combat.ID,
		ctx.PlayerName,
		targetID,
	)
	if err != nil {
		return mapCombatError(err), nil
	}

	publishDomainEvents(ctx.Events, domainEvents)

	return protocol.OK(
		"target=" + targetID +
			" damage=" + strconv.Itoa(damage) +
			" combat=" + result.ID,
	), nil
}

func (c *CombatCommand) handleDefend(ctx *Context) (protocol.Message, error) {
	if len(ctx.RawArgs) != 1 {
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
	}

	combat, err := ctx.World.GetActiveCombatForPlayer(ctx.PlayerName)
	if err != nil {
		return mapCombatError(err), nil
	}

	result, domainEvents, err := ctx.World.DefendCombatParticipant(
		combat.ID,
		ctx.PlayerName,
	)
	if err != nil {
		return mapCombatError(err), nil
	}

	publishDomainEvents(ctx.Events, domainEvents)

	return protocol.OK(
		"defended=true" +
			" defense_bonus=" + strconv.Itoa(world.DefaultDefenseBonusPercent) +
			" combat=" + result.ID,
	), nil
}

func (c *CombatCommand) handleUse(ctx *Context) (protocol.Message, error) {
	if len(ctx.RawArgs) < 2 {
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
	}

	itemName := strings.TrimSpace(strings.Join(ctx.RawArgs[1:], " "))
	if itemName == "" {
		return protocol.ERR(protocol.CodeInvalidArguments, "ITEM_REQUIRED"), nil
	}

	combat, err := ctx.World.GetActiveCombatForPlayer(ctx.PlayerName)
	if err != nil {
		return mapCombatError(err), nil
	}

	inventory, err := ctx.World.GetInventory(ctx.PlayerName)
	if err != nil {
		return mapCombatError(err), nil
	}

	var itemID string
	for _, item := range inventory {
		if strings.EqualFold(item.ID, itemName) || strings.EqualFold(item.Name, itemName) {
			if itemID != "" && itemID != item.ID {
				return protocol.ERR(protocol.CodeInvalidArguments, "AMBIGUOUS_ITEM_NAME"), nil
			}
			itemID = item.ID
		}
	}

	if itemID == "" {
		return protocol.ERR(protocol.CodeItemNotInInventory, "ITEM_NOT_IN_INVENTORY"), nil
	}

	// Free action (0 speed points): does not advance turns.
	result, event, err := ctx.World.CombatUseItem(combat.ID, ctx.PlayerName, itemID)
	if err != nil {
		return mapCombatError(err), nil
	}

	if ctx.Events != nil {
		ctx.Events.Publish(event)
	}

	stats := make([]string, 0, len(event.StatChanges))
	for stat := range event.StatChanges {
		stats = append(stats, stat)
	}
	sort.Strings(stats)

	var changes []string
	for _, stat := range stats {
		changes = append(changes, stat+"="+strconv.Itoa(event.StatChanges[stat]))
	}

	response := "used=" + event.ItemID
	if len(changes) > 0 {
		response += " changes=" + strings.Join(changes, ",")
	}
	response += " combat=" + result.ID

	return protocol.OK(response), nil
}

func (c *CombatCommand) handleFlee(ctx *Context) (protocol.Message, error) {
	if len(ctx.RawArgs) != 1 {
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
	}

	combat, err := ctx.World.GetActiveCombatForPlayer(ctx.PlayerName)
	if err != nil {
		return mapCombatError(err), nil
	}

	result, damage, remainingHP, domainEvents, err := ctx.World.FleeCombat(
		combat.ID,
		ctx.PlayerName,
	)
	if err != nil {
		return mapCombatError(err), nil
	}

	publishDomainEvents(ctx.Events, domainEvents)

	return protocol.OK(
		"fled=true" +
			" damage=" + strconv.Itoa(damage) +
			" hp=" + strconv.Itoa(remainingHP) +
			" combat=" + result.ID,
	), nil
}

func (c *CombatCommand) handleHelp(ctx *Context) (protocol.Message, error) {
	if len(ctx.RawArgs) != 1 {
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
	}

	result, event, err := ctx.World.RequestCombatHelp(ctx.PlayerName)
	if err != nil {
		return mapCombatError(err), nil
	}

	if event != nil && ctx.Events != nil {
		ctx.Events.Publish(*event)
	}

	return protocol.OK("combat=" + result.ID), nil
}

func (c *CombatCommand) handleJoin(ctx *Context) (protocol.Message, error) {
	if len(ctx.RawArgs) != 2 {
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
	}

	combatID := strings.TrimSpace(ctx.RawArgs[1])
	if combatID == "" {
		return protocol.ERR(protocol.CodeInvalidArguments, "INVALID_ARGUMENTS"), nil
	}

	result, domainEvents, err := ctx.World.JoinCombat(ctx.PlayerName, combatID)
	if err != nil {
		return mapCombatError(err), nil
	}

	publishDomainEvents(ctx.Events, domainEvents)

	return protocol.OK("combat=" + result.ID), nil
}