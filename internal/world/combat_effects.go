package world

import (
	"fmt"
	"strings"
)

// ApplyCombatEffect applies a generic temporary effect to a combat participant.
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
	effect.TargetID = strings.TrimSpace(effect.TargetID)
	effect.Stat = strings.ToLower(strings.TrimSpace(effect.Stat))

	if combatID == "" || participantID == "" ||
		effect.ID == "" || effect.Name == "" ||
		effect.TargetID == "" || effect.Stat == "" {
		return nil, fmt.Errorf(
			"%w: effect identifiers and stat cannot be empty",
			ErrInvalidArgument,
		)
	}

	if effect.TargetID != participantID {
		return nil, fmt.Errorf(
			"%w: effect target does not match participant",
			ErrInvalidArgument,
		)
	}

	if effect.Duration <= 0 || effect.RemainingTurns <= 0 ||
		effect.RemainingTurns > effect.Duration {
		return nil, fmt.Errorf(
			"%w: invalid effect duration",
			ErrInvalidArgument,
		)
	}

	if effect.Modifier == 0 && effect.ModifierPercent == 0 {
		return nil, fmt.Errorf(
			"%w: effect must have a modifier",
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

// calculateEffectiveStat returns a participant's stat
// after applying all active effects targeting that stat.
func calculateEffectiveStat(
	participant CombatParticipant,
	stat string,
) int {
	stat = strings.ToLower(strings.TrimSpace(stat))

	var baseStat int

	switch stat {
	case CombatStatHP:
		baseStat = participant.Stats.HP
	case CombatStatAttack:
		baseStat = participant.Stats.Attack
	case CombatStatDefense:
		baseStat = participant.Stats.Defense
	case CombatStatSpeed:
		baseStat = participant.Stats.Speed
	default:
		return 0
	}

	flatModifier := 0
	percentModifier := 0

	for _, effect := range participant.Effects {
		if effect.TargetID != participant.ID ||
			effect.Stat != stat {
			continue
		}

		flatModifier += effect.Modifier
		percentModifier += effect.ModifierPercent
	}

	result := baseStat + flatModifier
	result += result * percentModifier / 100

	if result < 0 {
		return 0
	}

	return result
}


// calculateEffectiveSpeed returns the participant's effective speed.
func calculateEffectiveSpeed(participant CombatParticipant) int {
	return calculateEffectiveStat(participant, CombatStatSpeed)
}

// updateCombatEffectsLocked updates effects belonging to the participant
// whose turn is starting.
// The caller must hold w.mu.
func (w *World) updateCombatEffectsLocked(
	combat *Combat,
	participantID string,
) (*Combat, error) {
	if combat == nil || !combat.Active {
		return nil, fmt.Errorf(
			"%w: inactive combat",
			ErrInvalidWorldState,
		)
	}

	if strings.TrimSpace(participantID) == "" {
		return nil, fmt.Errorf(
			"%w: participant ID cannot be empty",
			ErrInvalidArgument,
		)
	}

	for i := range combat.Participants {
		participant := &combat.Participants[i]

		if participant.ID != participantID {
			continue
		}

		if !participant.Alive || participant.Escaped {
			return nil, fmt.Errorf(
				"%w: participant %q cannot update effects",
				ErrInvalidWorldState,
				participantID,
			)
		}

		activeEffects := participant.Effects[:0]

		for _, effect := range participant.Effects {
			effect.RemainingTurns--

			if effect.RemainingTurns > 0 {
				activeEffects = append(activeEffects, effect)
			}
		}

		participant.Effects = activeEffects

		return combat, nil
	}

	return nil, fmt.Errorf(
		"%w: participant %q not found in combat",
		ErrInvalidWorldState,
		participantID,
	)
}