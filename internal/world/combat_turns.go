package world

import (
	"fmt"
	"strings"
)

// AvailableActions returns the number of available actions for a participant.
func AvailableActions(participant CombatParticipant) int {
	if !participant.Alive ||
		participant.Escaped ||
		(participant.IsPlayer && !participant.Connected) ||
		CombatActionCost <= 0 {
		return 0
	}

	return participant.SpeedPoints / CombatActionCost
}

// initializeCombatTurnsLocked initializes the first combat round on an active combat.
// The caller must hold w.mu.Lock().
func (w *World) initializeCombatTurnsLocked(
	combat *Combat,
) (*CombatTurnChanged, error) {
	if combat == nil || !combat.Active {
		return nil, fmt.Errorf("%w: active combat is required", ErrInvalidWorldState)
	}

	if combat.TurnsInitialized {
		return nil, fmt.Errorf("%w: combat turns have already been initialized", ErrInvalidWorldState)
	}

	if !combatHasLivingPlayersLocked(combat) {
		return nil, fmt.Errorf("%w: combat has no living players", ErrInvalidWorldState)
	}

	if !combatHasLivingNPCsLocked(combat) {
		return nil, fmt.Errorf("%w: all NPCs have been defeated", ErrInvalidWorldState)
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
		return nil, fmt.Errorf("%w: combat has no eligible participants", ErrInvalidWorldState)
	}

	combat.TurnIndex = firstTurnIndex
	combat.TurnsInitialized = true

	return w.buildCombatTurnChangedLocked(combat), nil
}

// GetCurrentCombatTurn returns the current participant's turn event.
func (w *World) GetCurrentCombatTurn(combatID string) (*CombatTurnChanged, error) {
	if w == nil {
		return nil, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	combatID = strings.TrimSpace(combatID)
	if combatID == "" {
		return nil, ErrInvalidArgument
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	combat, exists := w.combats[combatID]
	if !exists || combat == nil || !combat.Active || !combat.TurnsInitialized {
		return nil, fmt.Errorf("%w: active combat %q not found", ErrInvalidWorldState, combatID)
	}

	if combat.TurnIndex < 0 || combat.TurnIndex >= len(combat.Participants) {
		return nil, fmt.Errorf("%w: invalid combat turn index", ErrInvalidWorldState)
	}

	return w.buildCombatTurnChangedLocked(combat), nil
}

// SkipPlayerCombatTurn consumes a player's remaining actions and resolves
// all subsequent NPC turns or round transitions atomically.
func (w *World) SkipPlayerCombatTurn(
	combatID string,
	participantID string,
) ([]any, error) {
	if w == nil {
		return nil, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	combatID = strings.TrimSpace(combatID)
	participantID = strings.TrimSpace(participantID)
	if combatID == "" || participantID == "" {
		return nil, fmt.Errorf("%w: combat and participant IDs are required", ErrInvalidArgument)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	combat, err := w.getActiveCombatLocked(combatID)
	if err != nil {
		return nil, err
	}

	if combat.TurnIndex < 0 || combat.TurnIndex >= len(combat.Participants) {
		return nil, ErrInvalidCombatTurn
	}

	participant := combat.Participants[combat.TurnIndex]
	if !participant.IsPlayer || participant.ID != participantID {
		return nil, ErrInvalidCombatTurn
	}

	turnEvent, err := w.skipRemainingCombatActionsLocked(combat)
	if err != nil {
		return nil, err
	}

	events := make([]any, 0, 8)
	if turnEvent != nil {
		events = append(events, *turnEvent)
	}

	progressionEvents, err := w.resolveCombatProgressionLocked(combat)
	if err != nil {
		return nil, err
	}
	events = append(events, progressionEvents...)

	return events, nil
}

// consumeCombatActionLocked consumes one action from the current participant.
// The caller must hold w.mu.Lock().
func (w *World) consumeCombatActionLocked(combat *Combat) error {
	if combat == nil || !combat.Active || !combat.TurnsInitialized {
		return fmt.Errorf("%w: invalid combat state", ErrInvalidWorldState)
	}

	if len(combat.Participants) == 0 ||
		combat.TurnIndex < 0 ||
		combat.TurnIndex >= len(combat.Participants) {
		return fmt.Errorf("%w: invalid combat turn index", ErrInvalidWorldState)
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
// The caller must hold w.mu.Lock().
func (w *World) skipRemainingCombatActionsLocked(
	combat *Combat,
) (*CombatTurnChanged, error) {
	if combat == nil || !combat.Active || !combat.TurnsInitialized {
		return nil, fmt.Errorf("%w: invalid combat state", ErrInvalidWorldState)
	}

	if len(combat.Participants) == 0 ||
		combat.TurnIndex < 0 ||
		combat.TurnIndex >= len(combat.Participants) {
		return nil, fmt.Errorf("%w: invalid combat turn index", ErrInvalidWorldState)
	}

	participantIndex := combat.TurnIndex
	participant := &combat.Participants[participantIndex]

	if !participant.Alive ||
		(participant.IsPlayer && !participant.Connected) ||
		participant.SpeedPoints < CombatActionCost {
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
// The caller must hold w.mu.Lock().
func (w *World) advanceCombatTurnLocked(combat *Combat) (bool, error) {
	if combat == nil || len(combat.Participants) == 0 {
		if combat != nil {
			combat.TurnIndex = -1
		}
		return false, nil
	}

	if combat.TurnIndex < -1 || combat.TurnIndex >= len(combat.Participants) {
		return false, fmt.Errorf("%w: invalid combat turn index", ErrInvalidWorldState)
	}

	previousParticipants := make([]CombatParticipant, len(combat.Participants))
	for i, participant := range combat.Participants {
		previousParticipants[i] = participant
		previousParticipants[i].Effects = append([]CombatEffect(nil), participant.Effects...)
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
// The caller must hold w.mu.Lock().
func (w *World) buildCombatTurnChangedLocked(combat *Combat) *CombatTurnChanged {
	if combat == nil || len(combat.Participants) == 0 {
		return nil
	}

	if combat.TurnIndex < 0 || combat.TurnIndex >= len(combat.Participants) {
		return nil
	}

	participant := combat.Participants[combat.TurnIndex]
	availableActions := AvailableActions(participant)
	if availableActions <= 0 {
		return nil
	}

	return &CombatTurnChanged{
		CombatID:         combat.ID,
		ParticipantID:    participant.ID,
		ParticipantName:  participant.Name,
		IsPlayer:         participant.IsPlayer,
		Round:            combat.Round,
		AvailableActions: availableActions,
		Recipients:       w.getCombatRecipientsLocked(combat),
	}
}