package world

import (
	"fmt"
	"strings"
)

// AttackCombatParticipant executes an attack against another combat participant.
func (w *World) AttackCombatParticipant(
	combatID string,
	attackerID string,
	targetID string,
) (*Combat, int, *CombatAttack, *CombatDamage, *ParticipantDied, error) {
	if w == nil {
		return nil, 0, nil, nil, nil, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	combatID = strings.TrimSpace(combatID)
	attackerID = strings.TrimSpace(attackerID)
	targetID = strings.TrimSpace(targetID)

	if combatID == "" || attackerID == "" || targetID == "" {
		return nil, 0, nil, nil, nil, ErrInvalidArgument
	}

	if attackerID == targetID {
		return nil, 0, nil, nil, nil, ErrInvalidCombatTarget
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	combat, err := w.getActiveCombatLocked(combatID)
	if err != nil {
		return nil, 0, nil, nil, nil, err
	}

	attacker, err := w.getCurrentAttackerLocked(combat, attackerID)
	if err != nil {
		return nil, 0, nil, nil, nil, err
	}

	target, err := w.getCombatTargetLocked(
		combat,
		targetID,
		attacker.IsPlayer,
	)
	if err != nil {
		return nil, 0, nil, nil, nil, err
	}

	// Keep the target index because advancing the turn can restore
	// the Participants slice during rollback.
	targetIndex := -1

	for i := range combat.Participants {
		if combat.Participants[i].ID == targetID {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		return nil, 0, nil, nil, nil, ErrInvalidCombatTarget
	}

	// Calculate damage using effective stats.
	damage := calculateDamage(
		calculateEffectiveStat(*attacker, CombatStatAttack),
		calculateEffectiveStat(*target, CombatStatDefense),
	)

	// Preserve the target state in case action consumption fails.
	previousHP := target.Stats.HP
	previousAlive := target.Alive

	var previousEntityHP int

	if target.IsPlayer {
		player := w.players[target.ID]
		if player == nil {
			return nil, 0, nil, nil, nil, ErrPlayerNotFound
		}
		previousEntityHP = player.CombatStats.HP
	} else {
		npc, exists := w.npcs[target.ID]
		if !exists {
			return nil, 0, nil, nil, nil, ErrNPCNotFound
		}
		previousEntityHP = npc.CombatStats.HP
	}

	damage, err = w.applyCombatDamageLocked(target, damage)
	if err != nil {
		return nil, 0, nil, nil, nil, err
	}

	// Consume one action and advance the turn when necessary.
	if err := w.consumeCombatActionLocked(combat); err != nil {
		// Restore the target participant.
		combat.Participants[targetIndex].Stats.HP = previousHP
		combat.Participants[targetIndex].Alive = previousAlive

		// Restore the entity's HP.
		if target.IsPlayer {
			w.players[targetID].CombatStats.HP = previousEntityHP
		} else {
			npc := w.npcs[targetID]
			npc.CombatStats.HP = previousEntityHP
			w.npcs[targetID] = npc
		}

		return nil, 0, nil, nil, nil, err
	}

	// Retrieve the target again after consuming the action.
	target = &combat.Participants[targetIndex]

	recipients := w.getCombatRecipientsLocked(combat)

	attackEvent := &CombatAttack{
		CombatID:     combat.ID,
		AttackerID:   attacker.ID,
		AttackerName: attacker.Name,
		TargetID:     target.ID,
		TargetName:   target.Name,
		IsPlayer:     attacker.IsPlayer,
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

	return cloneCombat(combat), damage,
		attackEvent, damageEvent, deathEvent, nil
}

// getActiveCombatLocked retrieves an active combat.
// The caller must hold w.mu.
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

// getCurrentAttackerLocked retrieves the participant whose turn is active.
// The caller must hold w.mu.
func (w *World) getCurrentAttackerLocked(
	combat *Combat,
	attackerID string,
) (*CombatParticipant, error) {
	if combat == nil ||
		len(combat.Participants) == 0 ||
		combat.TurnIndex < 0 ||
		combat.TurnIndex >= len(combat.Participants) {
		return nil, ErrInvalidWorldState
	}

	attacker := &combat.Participants[combat.TurnIndex]

	if attacker.ID != attackerID ||
		!attacker.Alive ||
		attacker.Escaped ||
		AvailableActions(*attacker) <= 0 {
		return nil, ErrInvalidCombatTurn
	}

	return attacker, nil
}

// getCombatTargetLocked retrieves a valid opposing combat participant.
// The caller must hold w.mu.
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
			player, exists := w.players[target.ID]
			if !exists || player == nil {
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

// applyCombatDamageLocked applies damage and synchronizes the entity's HP.
// The caller must hold w.mu.
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

	// Preserve minimum damage.
	if damage < MinimumDamage {
		damage = MinimumDamage
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