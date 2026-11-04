package world

import (
	"fmt"
	"math/rand"
	"strings"
)

// ExecuteNPCCombatTurn executes the current NPC's combat action.
func (w *World) ExecuteNPCCombatTurn(
	combatID string,
) (*Combat, int, *CombatAttack, *CombatDamage, *ParticipantDied, error) {
	if w == nil {
		return nil, 0, nil, nil, nil, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	combatID = strings.TrimSpace(combatID)
	if combatID == "" {
		return nil, 0, nil, nil, nil, ErrInvalidArgument
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	combat, err := w.getActiveCombatLocked(combatID)
	if err != nil {
		return nil, 0, nil, nil, nil, err
	}

	if len(combat.Participants) == 0 ||
		combat.TurnIndex < 0 ||
		combat.TurnIndex >= len(combat.Participants) {
		return nil, 0, nil, nil, nil, ErrInvalidWorldState
	}

	// Get the current participant.
	attacker := &combat.Participants[combat.TurnIndex]

	// Only NPCs can execute this method.
	if attacker.IsPlayer ||
		!attacker.Alive ||
		attacker.Escaped ||
		AvailableActions(*attacker) <= 0 {
		return nil, 0, nil, nil, nil, ErrInvalidCombatTurn
	}

	if _, exists := w.npcs[attacker.ID]; !exists {
		return nil, 0, nil, nil, nil, ErrNPCNotFound
	}

	// Select a player target using the combat AI priorities.
	target, err := selectNPCCombatTarget(combat)
	if err != nil {
		return nil, 0, nil, nil, nil, err
	}

	// Keep the target index because advancing the turn can restore
	// the Participants slice during rollback.
	targetIndex := -1

	for i := range combat.Participants {
		if combat.Participants[i].ID == target.ID {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		return nil, 0, nil, nil, nil, ErrInvalidCombatTarget
	}

	// Preserve the target state in case action consumption fails.
	previousHP := target.Stats.HP
	previousAlive := target.Alive

	player, exists := w.players[target.ID]
	if !exists || player == nil {
		return nil, 0, nil, nil, nil, ErrPlayerNotFound
	}

	previousEntityHP := player.CombatStats.HP

	// Calculate damage using effective stats.
	damage := calculateDamage(
		calculateEffectiveStat(*attacker, CombatStatAttack),
		calculateEffectiveStat(*target, CombatStatDefense),
	)

	damage, err = w.applyCombatDamageLocked(target, damage)
	if err != nil {
		return nil, 0, nil, nil, nil, err
	}

	recipients := w.getCombatRecipientsLocked(combat)

	attackEvent := &CombatAttack{
		CombatID:     combat.ID,
		AttackerID:   attacker.ID,
		AttackerName: attacker.Name,
		TargetID:     target.ID,
		TargetName:   target.Name,
		IsPlayer:     false,
		Recipients:   recipients,
	}

	damageEvent := &CombatDamage{
		CombatID:     combat.ID,
		AttackerID:   attacker.ID,
		AttackerName: attacker.Name,
		TargetID:     target.ID,
		TargetName:   target.Name,
		Damage:       damage,
		RemainingHP:  target.Stats.HP,
		Recipients:   recipients,
	}

	var deathEvent *ParticipantDied

	if !target.Alive {
		deathEvent = &ParticipantDied{
			CombatID:        combat.ID,
			ParticipantID:   target.ID,
			ParticipantName: target.Name,
			IsPlayer:        target.IsPlayer,
			KillerID:        attacker.ID,
			Recipients:      recipients,
		}
	}

	// Consume one NPC action.
	if err := w.consumeCombatActionLocked(combat); err != nil {
		// Restore the target participant.
		combat.Participants[targetIndex].Stats.HP = previousHP
		combat.Participants[targetIndex].Alive = previousAlive

		// Restore the player's HP.
		player := w.players[target.ID]
		player.CombatStats.HP = previousEntityHP

		return nil, 0, nil, nil, nil, err
	}

	return cloneCombat(combat), damage,
		attackEvent, damageEvent, deathEvent, nil
}

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