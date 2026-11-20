package world

import (
	"fmt"
	"strings"
)

// FleeCombat allows a player to escape from an active combat.
// The player cannot rejoin this combat, may join other combats,
// and receives no further combat events after fleeing.
func (w *World) FleeCombat(
	combatID string,
	playerID string,
) (*Combat, int, int, []any, error) {
	if w == nil {
		return nil, 0, 0, nil, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	combatID = strings.TrimSpace(combatID)
	playerID = strings.TrimSpace(playerID)
	if combatID == "" || playerID == "" {
		return nil, 0, 0, nil, ErrInvalidArgument
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	combat, err := w.getActiveCombatLocked(combatID)
	if err != nil {
		return nil, 0, 0, nil, err
	}

	participant, err := w.getCurrentAttackerLocked(combat, playerID)
	if err != nil {
		return nil, 0, 0, nil, err
	}

	if !participant.IsPlayer {
		return nil, 0, 0, nil, ErrInvalidCombatTurn
	}

	player, exists := w.players[playerID]
	if !exists || player == nil {
		return nil, 0, 0, nil, ErrPlayerNotFound
	}

	damage := participant.Stats.HP * FleeHealthPenaltyPercent / 100
	if damage < MinimumDamage {
		damage = MinimumDamage
	}

	maxDamage := participant.Stats.HP - 1
	if damage > maxDamage {
		damage = maxDamage
	}

	participant.Stats.HP -= damage
	player.CombatStats.HP = participant.Stats.HP
	participant.Escaped = true
	remainingHP := participant.Stats.HP

	events := make([]any, 0, 8)
	events = append(events, PlayerFled{
		CombatID:    combat.ID,
		PlayerID:    participant.ID,
		PlayerName:  participant.Name,
		Damage:      damage,
		RemainingHP: remainingHP,
		Recipients:  w.getCombatRecipientsLocked(combat),
	})

	turnEvent, err := w.skipRemainingCombatActionsLocked(combat)
	if err != nil {
		return nil, 0, 0, nil, err
	}
	if turnEvent != nil {
		events = append(events, *turnEvent)
	}

	progressionEvents, err := w.resolveCombatProgressionLocked(combat)
	if err != nil {
		return nil, 0, 0, nil, err
	}
	events = append(events, progressionEvents...)

	return cloneCombat(combat), damage, remainingHP, events, nil
}