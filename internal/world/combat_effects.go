package world

import (
	"fmt"
	"strings"
)

// ApplyCombatEffect applies a temporary effect to a combat participant.
func (w *World) ApplyCombatEffect(
	combatID string,
	participantID string,
	effect CombatEffect,
) (*Combat, error) {
	if w == nil {
		return nil, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	combatID = strings.TrimSpace(combatID)
	participantID = strings.TrimSpace(participantID)
	effect.ID = strings.TrimSpace(effect.ID)
	effect.Name = strings.TrimSpace(effect.Name)

	if combatID == "" || participantID == "" ||
		effect.ID == "" || effect.Name == "" {
		return nil, fmt.Errorf(
			"%w: combat, participant and effect identifiers cannot be empty",
			ErrInvalidArgument,
		)
	}

	if effect.RemainingRounds <= 0 {
		return nil, fmt.Errorf(
			"%w: effect duration must be greater than zero",
			ErrInvalidArgument,
		)
	}

	if effect.SpeedModifier == 0 {
		return nil, fmt.Errorf(
			"%w: speed modifier cannot be zero",
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

	for i := range combat.Participants {
		participant := &combat.Participants[i]

		if participant.ID != participantID {
			continue
		}

		if !participant.Alive || participant.Escaped {
			return nil, fmt.Errorf(
				"%w: participant %q cannot receive effects",
				ErrInvalidWorldState,
				participantID,
			)
		}

		for _, existing := range participant.Effects {
			if existing.ID == effect.ID {
				return nil, fmt.Errorf(
					"%w: effect %q already exists",
					ErrInvalidWorldState,
					effect.ID,
				)
			}
		}

		participant.Effects = append(participant.Effects, effect)

		return cloneCombat(combat), nil
	}

	return nil, fmt.Errorf(
		"%w: participant %q not found in combat",
		ErrInvalidWorldState,
		participantID,
	)
}

// RemoveCombatEffect removes an effect from a combat participant.
func (w *World) RemoveCombatEffect(
	combatID string,
	participantID string,
	effectID string,
) (*Combat, error) {
	if w == nil {
		return nil, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	combatID = strings.TrimSpace(combatID)
	participantID = strings.TrimSpace(participantID)
	effectID = strings.TrimSpace(effectID)

	if combatID == "" || participantID == "" || effectID == "" {
		return nil, fmt.Errorf(
			"%w: combat, participant and effect identifiers cannot be empty",
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

	for i := range combat.Participants {
		participant := &combat.Participants[i]

		if participant.ID != participantID {
			continue
		}

		for j := range participant.Effects {
			if participant.Effects[j].ID != effectID {
				continue
			}

			participant.Effects = append(
				participant.Effects[:j],
				participant.Effects[j+1:]...,
			)

			return cloneCombat(combat), nil
		}

		return nil, fmt.Errorf(
			"%w: effect %q not found on participant %q",
			ErrInvalidWorldState,
			effectID,
			participantID,
		)
	}

	return nil, fmt.Errorf(
		"%w: participant %q not found in combat",
		ErrInvalidWorldState,
		participantID,
	)
}

// UpdateCombatEffects reduces the remaining duration of active effects.
// Effects reaching zero rounds are removed.
func (w *World) UpdateCombatEffects(combatID string) (*Combat, error) {
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

	updatedCombat, err := w.updateCombatEffectsLocked(combat)
	if err != nil {
		return nil, err
	}

	return cloneCombat(updatedCombat), nil
}

// calculateEffectiveSpeed returns the participant's speed
// after applying all active speed modifiers.
func calculateEffectiveSpeed(participant CombatParticipant) int {
	speed := participant.Stats.Speed

	for _, effect := range participant.Effects {
		speed += effect.SpeedModifier
	}

	if speed < 0 {
		return 0
	}

	return speed
}

func (w *World) updateCombatEffectsLocked(combat *Combat) (*Combat, error) {
	if combat == nil || !combat.Active {
		return nil, fmt.Errorf("%w: inactive combat", ErrInvalidWorldState)
	}

	for i := range combat.Participants {
		participant := &combat.Participants[i]
		activeEffects := participant.Effects[:0]

		for _, effect := range participant.Effects {
			effect.RemainingRounds--

			if effect.RemainingRounds > 0 {
				activeEffects = append(activeEffects, effect)
			}
		}

		participant.Effects = activeEffects
	}

	return combat, nil
}