package world

import (
	"fmt"
	"strings"
)

// DefendCombatParticipant applies a defensive effect
// and ends the participant's turn.
func (w *World) DefendCombatParticipant(
	combatID string,
	participantID string,
) (*Combat, *CombatTurnChanged, error) {
	if w == nil {
		return nil, nil, ErrInvalidWorldState
	}

	combatID = strings.TrimSpace(combatID)
	participantID = strings.TrimSpace(participantID)

	if combatID == "" || participantID == "" {
		return nil, nil, ErrInvalidArgument
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	combat, exists := w.combats[combatID]
	if !exists || combat == nil || !combat.Active {
		return nil, nil, ErrInvalidWorldState
	}

	if !combat.TurnsInitialized ||
		combat.TurnIndex < 0 ||
		combat.TurnIndex >= len(combat.Participants) {
		return nil, nil, ErrInvalidWorldState
	}

	participantIndex := combat.TurnIndex
	participant := &combat.Participants[participantIndex]

	if participant.ID != participantID ||
		!participant.IsPlayer ||
		!participant.Alive ||
		!participant.Connected ||
		AvailableActions(*participant) <= 0 {
		return nil, nil, ErrInvalidCombatTurn
	}

	// Preserve the original state for rollback.
	previousEffects := append(
		[]CombatEffect(nil),
		participant.Effects...,
	)
	previousSpeedPoints := participant.SpeedPoints

	// Apply the defensive effect.
	effect := CombatEffect{
		ID: fmt.Sprintf(
			"defend_%s_%d_%d",
			participantID,
			combat.Round,
			len(participant.Effects),
		),
		Name:            "Defend",
		TargetID:        participantID,
		Stat:            CombatStatDefense,
		ModifierPercent: DefaultDefenseBonusPercent,
		Duration:        1,
		RemainingTurns:  1,
	}

	participant.Effects = append(participant.Effects, effect)

	// End all remaining actions and advance the turn.
	event, err := w.skipRemainingCombatActionsLocked(combat)
	if err != nil {
		combat.Participants[participantIndex].Effects = previousEffects
		combat.Participants[participantIndex].SpeedPoints = previousSpeedPoints
		return nil, nil, err
	}

	return cloneCombat(combat), event, nil
}