package world

import (
	"math/rand"
)
// selectNPCCombatTarget selects a connected, living player participant.
//
// Priority:
// 1. Lowest remaining HP.
// 2. Closest position to the current turn.
// 3. Random selection when a tie remains.
func selectNPCCombatTarget(
	combat *Combat,
) (*CombatParticipant, error) {
	if combat == nil ||
		len(combat.Participants) == 0 ||
		combat.TurnIndex < 0 ||
		combat.TurnIndex >= len(combat.Participants) {
		return nil, ErrInvalidCombatTarget
	}

	var candidates []*CombatParticipant
	minHP := int(^uint(0) >> 1)

	// Find living, connected player participants with the lowest HP.
	for i := range combat.Participants {
		participant := &combat.Participants[i]

		if !participant.IsPlayer ||
			!participant.Alive ||
			participant.Escaped ||
			!participant.Connected {
			continue
		}

		if participant.Stats.HP < minHP {
			minHP = participant.Stats.HP
			candidates = candidates[:0]
		}

		if participant.Stats.HP == minHP {
			candidates = append(candidates, participant)
		}
	}

	if len(candidates) == 0 {
		return nil, ErrInvalidCombatTarget
	}

	// Prefer the candidate whose turn is closest to the current TurnIndex.
	if len(candidates) > 1 {
		closest := len(combat.Participants)
		var closestCandidates []*CombatParticipant

		for i := range combat.Participants {
			participant := &combat.Participants[i]

			for _, candidate := range candidates {
				if participant.ID != candidate.ID {
					continue
				}

				distance := (i - combat.TurnIndex +
					len(combat.Participants)) % len(combat.Participants)

				if distance < closest {
					closest = distance
					closestCandidates = closestCandidates[:0]
				}

				if distance == closest {
					closestCandidates = append(
						closestCandidates,
						candidate,
					)
				}

				break
			}
		}

		candidates = closestCandidates
	}

	// Randomly resolve any remaining tie.
	return candidates[rand.Intn(len(candidates))], nil
}