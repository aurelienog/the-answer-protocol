package world

import (
	"fmt"
	"strings"
)

func (w *World) AttackCombatParticipant(
	combatID string,
	attackerID string,
	targetID string,
) (*Combat, int, *ParticipantDied, error) {
	if w == nil {
		return nil, 0, nil, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	combatID = strings.TrimSpace(combatID)
	attackerID = strings.TrimSpace(attackerID)
	targetID = strings.TrimSpace(targetID)

	if combatID == "" || attackerID == "" || targetID == "" {
		return nil, 0, nil, ErrInvalidArgument
	}

	if attackerID == targetID {
		return nil, 0, nil, ErrInvalidCombatTarget
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	combat, err := w.getActiveCombatLocked(combatID)
	if err != nil {
		return nil, 0, nil, err
	}

	attacker, err := w.getCurrentAttackerLocked(combat, attackerID)
	if err != nil {
		return nil, 0, nil, err
	}

	target, err := w.getCombatTargetLocked(combat, targetID, attacker.IsPlayer)
	if err != nil {
		return nil, 0, nil, err
	}

	damage := calculateDamage(
		attacker.Stats.Attack,
		target.Stats.Defense,
	)

	damage, err = w.applyCombatDamageLocked(target, damage)
	if err != nil {
		return nil, 0, nil, err
	}

	var deathEvent *ParticipantDied

	if !target.Alive {
		deathEvent = w.buildParticipantDiedEventLocked(
			combat,
			attacker,
			target,
		)
	}

	w.consumeCombatActionLocked(combat, attacker)

	return cloneCombat(combat), damage, deathEvent, nil
}

func (w *World) getActiveCombatLocked(
	combatID string,
) (*Combat, error) {
	combat, exists := w.combats[combatID]

	if !exists || combat == nil || !combat.Active {
		return nil, fmt.Errorf(
			"%w: active combat %q not found",
			ErrInvalidWorldState,
			combatID,
		)
	}

	if !combat.TurnsInitialized {
		return nil, fmt.Errorf(
			"%w: combat turns are not initialized",
			ErrInvalidWorldState,
		)
	}

	return combat, nil
}

func (w *World) getCurrentAttackerLocked(
	combat *Combat,
	attackerID string,
) (*CombatParticipant, error) {
	if len(combat.Participants) == 0 ||
		combat.TurnIndex < 0 ||
		combat.TurnIndex >= len(combat.Participants) {
		return nil, ErrInvalidWorldState
	}

	attacker := &combat.Participants[combat.TurnIndex]

	if attacker.ID != attackerID ||
		!attacker.Alive ||
		attacker.Escaped ||
		!attacker.Connected ||
		AvailableActions(*attacker) <= 0 {
		return nil, ErrInvalidCombatTurn
	}

	return attacker, nil
}

func (w *World) getCombatTargetLocked(
	combat *Combat,
	targetID string,
	attackerIsPlayer bool,
) (*CombatParticipant, error) {
	for i := range combat.Participants {
		target := &combat.Participants[i]

		if target.ID != targetID {
			continue
		}

		if !target.Alive ||
			target.Escaped ||
			target.IsPlayer == attackerIsPlayer {
			return nil, ErrInvalidCombatTarget
		}

		if target.IsPlayer {
			if _, exists := w.players[target.ID]; !exists {
				return nil, ErrPlayerNotFound
			}
		} else {
			if _, exists := w.npcs[target.ID]; !exists {
				return nil, ErrNPCNotFound
			}
		}

		return target, nil
	}

	return nil, ErrInvalidCombatTarget
}

func (w *World) applyCombatDamageLocked(
	target *CombatParticipant,
	damage int,
) (int, error) {
	if target == nil || damage < 0 {
		return 0, ErrInvalidArgument
	}

	if target.IsPlayer {
		player, exists := w.players[target.ID]
		if !exists || player == nil {
			return 0, ErrPlayerNotFound
		}
	} else {
		if _, exists := w.npcs[target.ID]; !exists {
			return 0, ErrNPCNotFound
		}
	}

	// Clamp damage to the target's remaining HP.
	if damage > target.Stats.HP {
		damage = target.Stats.HP
	}

	target.Stats.HP -= damage

	if target.Stats.HP <= 0 {
		target.Stats.HP = 0
		target.Alive = false
	}

	// Synchronize the entity's HP with the combat participant.
	if target.IsPlayer {
		w.players[target.ID].CombatStats.HP = target.Stats.HP
		return damage, nil
	}

	npc := w.npcs[target.ID]
	npc.CombatStats.HP = target.Stats.HP
	w.npcs[target.ID] = npc

	return damage, nil
}

func (w *World) consumeCombatActionLocked(
	combat *Combat,
	attacker *CombatParticipant,
) {
	attacker.SpeedPoints -= CombatActionCost

	if AvailableActions(*attacker) == 0 {
		combat.TurnIndex = (combat.TurnIndex + 1) % len(combat.Participants)
	}
}

func calculateDamage(attack, defense int) int {
	damage := attack - defense

	if damage < MinimumDamage {
		return MinimumDamage
	}

	return damage
}

func (w *World) buildParticipantDiedEventLocked(
	combat *Combat,
	attacker *CombatParticipant,
	target *CombatParticipant,
) *ParticipantDied {
	recipients := make([]string, 0)

	for _, participant := range combat.Participants {
		if participant.IsPlayer && participant.Connected {
			recipients = append(recipients, participant.ID)
		}
	}

	return &ParticipantDied{
		CombatID:        combat.ID,
		ParticipantID:   target.ID,
		ParticipantName: target.Name,
		IsPlayer:        target.IsPlayer,
		KillerID:        attacker.ID,
		Recipients:      recipients,
	}
}