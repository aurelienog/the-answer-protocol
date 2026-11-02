package world

import (
	"fmt"
	"math/rand"
	"strings"
)

// ExecuteNPCCombatTurn executes the current NPC's combat action.
func (w *World) ExecuteNPCCombatTurn(
	combatID string,
) (*Combat, int, *ParticipantDied, error) {
	if w == nil {
		return nil, 0, nil, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	combatID = strings.TrimSpace(combatID)
	if combatID == "" {
		return nil, 0, nil, fmt.Errorf(
			"%w: combat ID cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	combat, err := w.getActiveCombatLocked(combatID)
	if err != nil {
		return nil, 0, nil, err
	}

	if len(combat.Participants) == 0 {
		return nil, 0, nil, ErrInvalidWorldState
	}

	if combat.TurnIndex < 0 ||
		combat.TurnIndex >= len(combat.Participants) {
		return nil, 0, nil, fmt.Errorf(
			"%w: invalid combat turn index",
			ErrInvalidWorldState,
		)
	}

	// Get the current participant.
	attacker := &combat.Participants[combat.TurnIndex]

	// Only NPCs can execute this method.
	if attacker.IsPlayer {
		return nil, 0, nil, ErrInvalidCombatTurn
	}

	if !attacker.Alive || attacker.Escaped {
		return nil, 0, nil, ErrInvalidCombatTurn
	}

	if _, exists := w.npcs[attacker.ID]; !exists {
		return nil, 0, nil, ErrNPCNotFound
	}

	if AvailableActions(*attacker) <= 0 {
		return nil, 0, nil, ErrInvalidCombatTurn
	}

	// Select a player target using the combat AI priorities.
	target, err := selectNPCCombatTarget(combat)
	if err != nil {
		return nil, 0, nil, err
	}

	// Calculate and apply damage.
	damage := calculateDamage(
		attacker.Stats.Attack,
		target.Stats.Defense,
	)

	damage, err = w.applyCombatDamageLocked(target, damage)
	if err != nil {
		return nil, 0, nil, err
	}

	// Generate a death event if the target was defeated.
	var deathEvent *ParticipantDied

	if !target.Alive {
		deathEvent = w.buildParticipantDiedEventLocked(
			combat,
			attacker,
			target,
		)
	}

	// Consume one NPC action.
	w.consumeCombatActionLocked(combat, attacker)

	return cloneCombat(combat), damage, deathEvent, nil
}

func selectNPCCombatTarget(combat *Combat) (*CombatParticipant, error) {
	if combat == nil || len(combat.Participants) == 0 {
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
					closestCandidates = append(closestCandidates, candidate)
				}

				break
			}
		}

		candidates = closestCandidates
	}

	// Randomly resolve any remaining tie.
	return candidates[rand.Intn(len(candidates))], nil
}