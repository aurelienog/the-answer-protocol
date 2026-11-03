package world

import (
	"fmt"
	"strings"
)

// FleeCombat allows a player to escape from an active combat.
func (w *World) FleeCombat(
	combatID string,
	playerID string,
) (*Combat, *PlayerFled, error) {
	if w == nil {
		return nil, nil, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	combatID = strings.TrimSpace(combatID)
	playerID = strings.TrimSpace(playerID)

	if combatID == "" || playerID == "" {
		return nil, nil, ErrInvalidArgument
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	combat, err := w.getActiveCombatLocked(combatID)
	if err != nil {
		return nil, nil, err
	}

	participant, err := w.getCurrentAttackerLocked(combat, playerID)
	if err != nil {
		return nil, nil, err
	}

	if !participant.IsPlayer {
		return nil, nil, ErrInvalidCombatTurn
	}

	player, exists := w.players[playerID]
	if !exists || player == nil {
		return nil, nil, ErrPlayerNotFound
	}

	// Calculate the escape penalty using current HP.
	damage := participant.Stats.HP * FleeHealthPenaltyPercent / 100

	if damage < MinimumDamage {
		damage = MinimumDamage
	}

	// Never allow fleeing to kill the player.
	maxDamage := participant.Stats.HP - 1
	if damage > maxDamage {
		damage = maxDamage
	}

	participant.Stats.HP -= damage
	player.CombatStats.HP = participant.Stats.HP
	participant.Escaped = true

	// Consume all remaining action points.
	participant.SpeedPoints = 0

	// Advance the turn.
	combat.TurnIndex = (combat.TurnIndex + 1) % len(combat.Participants)

	event := &PlayerFled{
		CombatID:    combat.ID,
		PlayerID:    participant.ID,
		PlayerName:  participant.Name,
		Damage:      damage,
		RemainingHP: participant.Stats.HP,
		Recipients:  w.getCombatRecipientsLocked(combat),
	}

	return cloneCombat(combat), event, nil
}