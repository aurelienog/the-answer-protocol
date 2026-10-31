package world

import (
	"fmt"
	"strings"
)

// InitializeCombatTurns initializes the first combat round.
func (w *World) InitializeCombatTurns(combatID string) (*Combat, error) {
	if w == nil {
		return nil, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	combatID = strings.TrimSpace(combatID)

	if combatID == "" {
		return nil, fmt.Errorf(
			"%w: combat ID cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	combat, exists := w.combats[combatID]
	if !exists || combat == nil || !combat.Active {
		return nil, fmt.Errorf(
			"%w: active combat %q not found",
			ErrInvalidWorldState,
			combatID,
		)
	}

	if combat.Round != 1 {
		return nil, fmt.Errorf(
			"%w: combat turns have already been initialized",
			ErrInvalidWorldState,
		)
	}

	if !combatHasLivingPlayersLocked(combat) {
		return nil, fmt.Errorf(
			"%w: combat has no living players",
			ErrInvalidWorldState,
		)
	}

	if !combatHasLivingNPCsLocked(combat) {
		return nil, fmt.Errorf(
			"%w: all NPCs have been defeated",
			ErrInvalidWorldState,
		)
	}

	for i := range combat.Participants {
		participant := &combat.Participants[i]

		if !participant.Alive || participant.Escaped || !participant.Connected {
			continue
		}

		participant.SpeedPoints += calculateEffectiveSpeed(*participant)
	}

	return cloneCombat(combat), nil
}

// StartCombatRound starts the next combat round.
func (w *World) StartCombatRound(combatID string) (*Combat, error) {
	if w == nil {
		return nil, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	combatID = strings.TrimSpace(combatID)
	if combatID == "" {
		return nil, fmt.Errorf("%w: combat ID cannot be empty", ErrInvalidArgument)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	combat, exists := w.combats[combatID]
	if !exists || combat == nil || !combat.Active {
		return nil, fmt.Errorf("%w: active combat %q not found", ErrInvalidWorldState, combatID)
	}

	if !combatHasLivingPlayersLocked(combat) {
		return nil, fmt.Errorf(
			"%w: combat has no living players",
			ErrInvalidWorldState,
		)
	}

	if !combatHasLivingNPCsLocked(combat) {
		return nil, fmt.Errorf(
			"%w: all NPCs have been defeated",
			ErrInvalidWorldState,
		)
	}

	// Prevent advancing while participants still have available actions.
	if combatHasAvailableActionsLocked(combat) {
		return nil, fmt.Errorf(
			"%w: combat still has available actions",
			ErrInvalidWorldState,
		)
	}

	// Advance to the next round.
	combat.Round++

	// Accumulate speed using the effects active at the start of the round.
	for i := range combat.Participants {
		participant := &combat.Participants[i]

		if !participant.Alive || participant.Escaped || !participant.Connected {
			continue
		}

		participant.SpeedPoints += calculateEffectiveSpeed(*participant)
	}

	// Decrease effect durations after their contribution to this round.
	if _, err := w.updateCombatEffectsLocked(combat); err != nil {
		return nil, err
	}

	return cloneCombat(combat), nil
}


// AvailableActions returns the number of available actions.
func AvailableActions(participant CombatParticipant) int {
    if !participant.Alive || participant.Escaped || !participant.Connected {
        return 0
    }

    return participant.SpeedPoints / 10
}

// NextCombatTurn finds the next participant with available actions.
func (w *World) NextCombatTurn(combatID string) (*Combat, error) {
	if w == nil {
		return nil, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	combatID = strings.TrimSpace(combatID)
	if combatID == "" {
		return nil, fmt.Errorf("%w: combat ID cannot be empty", ErrInvalidArgument)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	combat, exists := w.combats[combatID]
	if !exists || combat == nil || !combat.Active {
		return nil, fmt.Errorf("%w: active combat %q not found", ErrInvalidWorldState, combatID)
	}

	if len(combat.Participants) == 0 {
		return nil, fmt.Errorf("%w: combat has no participants", ErrInvalidWorldState)
	}

	// Validate the current turn index.
	if combat.TurnIndex < 0 || combat.TurnIndex >= len(combat.Participants) {
		return nil, fmt.Errorf("%w: invalid combat turn index", ErrInvalidWorldState)
	}

	// Keep the current participant if they still have actions available.
	current := &combat.Participants[combat.TurnIndex]

	if AvailableActions(*current) > 0 {
		return cloneCombat(combat), nil
	}

	// Search circularly for the next participant with available actions.
	for offset := 1; offset <= len(combat.Participants); offset++ {
		index := (combat.TurnIndex + offset) % len(combat.Participants)
		participant := &combat.Participants[index]

		if AvailableActions(*participant) > 0 {
			combat.TurnIndex = index
			return cloneCombat(combat), nil
		}
	}

	return cloneCombat(combat), nil
}

// EndCombatTurn consumes one action from the current participant.
func (w *World) EndCombatTurn(combatID string) (*Combat, error) {
	if w == nil {
		return nil, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	combatID = strings.TrimSpace(combatID)
	if combatID == "" {
		return nil, fmt.Errorf("%w: combat ID cannot be empty", ErrInvalidArgument)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	combat, exists := w.combats[combatID]
	if !exists || combat == nil || !combat.Active {
		return nil, fmt.Errorf("%w: active combat %q not found", ErrInvalidWorldState, combatID)
	}

	if len(combat.Participants) == 0 ||
		combat.TurnIndex < 0 ||
		combat.TurnIndex >= len(combat.Participants) {
		return nil, fmt.Errorf("%w: invalid combat turn index", ErrInvalidWorldState)
	}

	participant := &combat.Participants[combat.TurnIndex]

	if AvailableActions(*participant) <= 0 {
		return nil, fmt.Errorf(
			"%w: participant %q has no available actions",
			ErrInvalidWorldState,
			participant.ID,
		)
	}

	participant.SpeedPoints -= 10

	// If the participant has no more actions, move to the next participant.
	if AvailableActions(*participant) == 0 {
		combat.TurnIndex = (combat.TurnIndex + 1) % len(combat.Participants)
	}

	return cloneCombat(combat), nil
}

func combatHasAvailableActionsLocked(combat *Combat) bool {
	if combat == nil {
		return false
	}

	for _, participant := range combat.Participants {
		if AvailableActions(participant) > 0 {
			return true
		}
	}

	return false
}

// IsCombatRoundComplete reports whether all active participants have exhausted their actions.
func (w *World) IsCombatRoundComplete(combatID string) (bool, error) {
	if w == nil {
		return false, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	combatID = strings.TrimSpace(combatID)
	if combatID == "" {
		return false, fmt.Errorf("%w: combat ID cannot be empty", ErrInvalidArgument)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	combat, exists := w.combats[combatID]
	if !exists || combat == nil || !combat.Active {
		return false, fmt.Errorf(
			"%w: active combat %q not found",
			ErrInvalidWorldState,
			combatID,
		)
	}

	if !combatHasLivingPlayersLocked(combat) {
		return false, fmt.Errorf(
			"%w: combat has no living players",
			ErrInvalidWorldState,
		)
	}

	if !combatHasLivingNPCsLocked(combat) {
		return false, fmt.Errorf(
			"%w: all NPCs have been defeated",
			ErrInvalidWorldState,
		)
	}

	return !combatHasAvailableActionsLocked(combat), nil
}