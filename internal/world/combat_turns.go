package world

import (
	"fmt"
	"strings"
)

// InitializeCombatTurns initializes the first combat round.
func (w *World) InitializeCombatTurns(
	combatID string,
) (*Combat, *CombatTurnChanged, error) {
	if w == nil {
		return nil, nil, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	combatID = strings.TrimSpace(combatID)
	if combatID == "" {
		return nil, nil, fmt.Errorf(
			"%w: combat ID cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	combat, exists := w.combats[combatID]
	if !exists || combat == nil || !combat.Active {
		return nil, nil, fmt.Errorf(
			"%w: active combat %q not found",
			ErrInvalidWorldState,
			combatID,
		)
	}

	if combat.TurnsInitialized {
		return nil, nil, fmt.Errorf(
			"%w: combat turns have already been initialized",
			ErrInvalidWorldState,
		)
	}

	if !combatHasLivingPlayersLocked(combat) {
		return nil, nil, fmt.Errorf(
			"%w: combat has no living players",
			ErrInvalidWorldState,
		)
	}

	if !combatHasLivingNPCsLocked(combat) {
		return nil, nil, fmt.Errorf(
			"%w: all NPCs have been defeated",
			ErrInvalidWorldState,
		)
	}

	firstTurnIndex := -1
	highestSpeed := 0

	for i := range combat.Participants {
		participant := &combat.Participants[i]

		if !participant.Alive || participant.Escaped {
			continue
		}

		if participant.IsPlayer && !participant.Connected {
			continue
		}

		effectiveSpeed := calculateEffectiveSpeed(*participant)
		participant.SpeedPoints += effectiveSpeed

		// Ties preserve the original participant order.
		if firstTurnIndex == -1 || effectiveSpeed > highestSpeed {
			highestSpeed = effectiveSpeed
			firstTurnIndex = i
		}
	}

	if firstTurnIndex == -1 {
		return nil, nil, fmt.Errorf(
			"%w: combat has no eligible participants",
			ErrInvalidWorldState,
		)
	}

	combat.TurnIndex = firstTurnIndex
	combat.TurnsInitialized = true

	event := w.buildCombatTurnChangedLocked(combat)

	return cloneCombat(combat), event, nil
}

// NextCombatTurn advances to the next participant when necessary.
func (w *World) NextCombatTurn(
	combatID string,
) (*Combat, *CombatTurnChanged, error) {
	if w == nil {
		return nil, nil, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	combatID = strings.TrimSpace(combatID)
	if combatID == "" {
		return nil, nil, fmt.Errorf(
			"%w: combat ID cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	combat, exists := w.combats[combatID]
	if !exists || combat == nil || !combat.Active {
		return nil, nil, fmt.Errorf(
			"%w: active combat %q not found",
			ErrInvalidWorldState,
			combatID,
		)
	}

	if !combat.TurnsInitialized {
		return nil, nil, fmt.Errorf(
			"%w: combat turns have not been initialized",
			ErrInvalidWorldState,
		)
	}

	if len(combat.Participants) == 0 {
		return nil, nil, fmt.Errorf(
			"%w: combat has no participants",
			ErrInvalidWorldState,
		)
	}

	if combat.TurnIndex < -1 ||
		combat.TurnIndex >= len(combat.Participants) {
		return nil, nil, fmt.Errorf(
			"%w: invalid combat turn index",
			ErrInvalidWorldState,
		)
	}

	// Keep the current turn if the participant can still act.
	if combat.TurnIndex >= 0 &&
		AvailableActions(combat.Participants[combat.TurnIndex]) > 0 {
		return cloneCombat(combat), nil, nil
	}

	advanced, err := w.advanceCombatTurnLocked(combat)
	if err != nil {
		return nil, nil, err
	}

	if !advanced {
		return cloneCombat(combat), nil, nil
	}

	event := w.buildCombatTurnChangedLocked(combat)

	return cloneCombat(combat), event, nil
}

// consumeCombatActionLocked consumes one action from the current participant.
// The caller must hold w.mu.
func (w *World) consumeCombatActionLocked(
	combat *Combat,
) error {
	if combat == nil || !combat.Active || !combat.TurnsInitialized {
		return fmt.Errorf(
			"%w: invalid combat state",
			ErrInvalidWorldState,
		)
	}

	if len(combat.Participants) == 0 ||
		combat.TurnIndex < 0 ||
		combat.TurnIndex >= len(combat.Participants) {
		return fmt.Errorf(
			"%w: invalid combat turn index",
			ErrInvalidWorldState,
		)
	}

	participantIndex := combat.TurnIndex
	participant := &combat.Participants[participantIndex]

	if AvailableActions(*participant) <= 0 {
		return ErrInvalidCombatTurn
	}

	previousSpeedPoints := participant.SpeedPoints
	participant.SpeedPoints -= CombatActionCost

	// Advance only when the participant has exhausted their actions.
	if AvailableActions(*participant) == 0 {
		if _, err := w.advanceCombatTurnLocked(combat); err != nil {
			combat.Participants[participantIndex].SpeedPoints = previousSpeedPoints
			return err
		}
	}

	return nil
}

// skipRemainingCombatActionsLocked consumes all remaining actions
// from the current participant and advances the turn.
// The caller must hold w.mu.
func (w *World) skipRemainingCombatActionsLocked(
	combat *Combat,
) (*CombatTurnChanged, error) {
	if combat == nil || !combat.Active || !combat.TurnsInitialized {
		return nil, fmt.Errorf(
			"%w: invalid combat state",
			ErrInvalidWorldState,
		)
	}

	if len(combat.Participants) == 0 ||
		combat.TurnIndex < 0 ||
		combat.TurnIndex >= len(combat.Participants) {
		return nil, fmt.Errorf(
			"%w: invalid combat turn index",
			ErrInvalidWorldState,
		)
	}

	participantIndex := combat.TurnIndex
	participant := &combat.Participants[participantIndex]

	if AvailableActions(*participant) <= 0 {
		return nil, ErrInvalidCombatTurn
	}

	previousSpeedPoints := participant.SpeedPoints
	participant.SpeedPoints = 0

	advanced, err := w.advanceCombatTurnLocked(combat)
	if err != nil {
		combat.Participants[participantIndex].SpeedPoints = previousSpeedPoints
		return nil, err
	}

	if !advanced {
		return nil, nil
	}

	return w.buildCombatTurnChangedLocked(combat), nil
}

// advanceCombatTurnLocked advances to the next participant with available actions.
// It updates the effects of each eligible participant before checking their actions.
// If an error occurs, it restores the original combat state.
// The caller must hold w.mu.
func (w *World) advanceCombatTurnLocked(
	combat *Combat,
) (bool, error) {
	if combat == nil || len(combat.Participants) == 0 {
		if combat != nil {
			combat.TurnIndex = -1
		}
		return false, nil
	}

	if combat.TurnIndex < -1 ||
		combat.TurnIndex >= len(combat.Participants) {
		return false, fmt.Errorf(
			"%w: invalid combat turn index",
			ErrInvalidWorldState,
		)
	}

	// Preserve the state in case advancing the turn fails.
	previousParticipants := make(
		[]CombatParticipant,
		len(combat.Participants),
	)

	for i, participant := range combat.Participants {
		previousParticipants[i] = participant
		previousParticipants[i].Effects = append(
			[]CombatEffect(nil),
			participant.Effects...,
		)
	}

	previousTurnIndex := combat.TurnIndex

	rollback := func() {
		combat.Participants = previousParticipants
		combat.TurnIndex = previousTurnIndex
	}

	currentIndex := combat.TurnIndex

	for offset := 1; offset <= len(combat.Participants); offset++ {
		index := (currentIndex + offset) % len(combat.Participants)
		participant := &combat.Participants[index]

		if !participant.Alive || participant.Escaped {
			continue
		}

		if participant.IsPlayer && !participant.Connected {
			continue
		}

		participantID := participant.ID
		combat.TurnIndex = index

		// Effects expire at the beginning of the participant's next turn.
		if _, err := w.updateCombatEffectsLocked(combat, participantID); err != nil {
			rollback()
			return false, err
		}

		// Refresh the pointer after updating effects.
		participant = &combat.Participants[index]

		if AvailableActions(*participant) <= 0 {
			continue
		}

		return true, nil
	}

	combat.TurnIndex = -1

	return false, nil
}

// buildCombatTurnChangedLocked creates a notification for the current turn.
// The caller must hold w.mu.
func (w *World) buildCombatTurnChangedLocked(
	combat *Combat,
) *CombatTurnChanged {
	if combat == nil || len(combat.Participants) == 0 {
		return nil
	}

	if combat.TurnIndex < 0 ||
		combat.TurnIndex >= len(combat.Participants) {
		return nil
	}

	participant := combat.Participants[combat.TurnIndex]
	availableActions := AvailableActions(participant)

	if availableActions <= 0 {
		return nil
	}

	return &CombatTurnChanged{
		CombatID:         combat.ID,
		ParticipantID:   participant.ID,
		ParticipantName: participant.Name,
		IsPlayer:         participant.IsPlayer,
		Round:            combat.Round,
		AvailableActions: availableActions,
		Recipients:       w.getCombatRecipientsLocked(combat),
	}
}

// AvailableActions returns the number of available actions.
func AvailableActions(participant CombatParticipant) int {
	if !participant.Alive ||
		participant.Escaped ||
		(participant.IsPlayer && !participant.Connected) ||
		CombatActionCost <= 0 {
		return 0
	}

	return participant.SpeedPoints / CombatActionCost
}