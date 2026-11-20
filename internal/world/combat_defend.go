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
) (*Combat, []any, error) {
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

	combat, err := w.getActiveCombatLocked(combatID)
	if err != nil {
		return nil, nil, err
	}

	if combat.TurnIndex < 0 || combat.TurnIndex >= len(combat.Participants) {
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

	previousEffects := append([]CombatEffect(nil), participant.Effects...)
	previousSpeedPoints := participant.SpeedPoints

	participant.Effects = append(participant.Effects, CombatEffect{
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
	})

	turnEvent, err := w.skipRemainingCombatActionsLocked(combat)
	if err != nil {
		combat.Participants[participantIndex].Effects = previousEffects
		combat.Participants[participantIndex].SpeedPoints = previousSpeedPoints
		return nil, nil, err
	}

	events := make([]any, 0, 8)
	if turnEvent != nil {
		events = append(events, *turnEvent)
	}

	progressionEvents, err := w.resolveCombatProgressionLocked(combat)
	if err != nil {
		return nil, nil, err
	}
	events = append(events, progressionEvents...)

	return cloneCombat(combat), events, nil
}