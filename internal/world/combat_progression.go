package world

import (
	"fmt"
)

// resolveCombatProgressionLocked advances turns, starts new rounds,
// executes consecutive NPC actions, and finishes combat when appropriate.
// The caller must hold w.mu.Lock().
func (w *World) resolveCombatProgressionLocked(combat *Combat) ([]any, error) {
	events := make([]any, 0)

	for {
		if combat == nil || !combat.Active {
			return events, nil
		}

		// 1. Check whether combat must end before taking any further turns.
		if !combatHasActivePlayersLocked(combat) || !combatHasLivingNPCsLocked(combat) {
			endEvent, respawnEvents, err := w.endCombatLocked(combat)
			if err != nil {
				return nil, err
			}
			if endEvent != nil {
				events = append(events, *endEvent)
			}
			for _, respawn := range respawnEvents {
				events = append(events, respawn)
			}
			return events, nil
		}

		// 2. Start a new round if all participants exhausted their actions,
		// or advance to the next participant if the current one cannot act.
		if !combatHasAvailableActionsLocked(combat) {
			turnEvent, err := w.startCombatRoundLocked(combat)
			if err != nil {
				return nil, err
			}
			if turnEvent != nil {
				events = append(events, *turnEvent)
			}
		} else if combat.TurnIndex < 0 ||
			combat.TurnIndex >= len(combat.Participants) ||
			AvailableActions(combat.Participants[combat.TurnIndex]) <= 0 {
			advanced, err := w.advanceCombatTurnLocked(combat)
			if err != nil {
				return nil, err
			}
			if advanced {
				if turnEvent := w.buildCombatTurnChangedLocked(combat); turnEvent != nil {
					events = append(events, *turnEvent)
				}
			}
		}

		if combat.TurnIndex < 0 || combat.TurnIndex >= len(combat.Participants) {
			return nil, ErrInvalidWorldState
		}

		current := combat.Participants[combat.TurnIndex]

		// 3. Stop when it is a player's turn.
		if current.IsPlayer {
			return events, nil
		}

		// 4. Execute the NPC's action.
		previousParticipantID := current.ID

		attackEvent, damageEvent, deathEvent, err := w.executeNPCCombatTurnLocked(combat)
		if err != nil {
			return nil, err
		}

		combatEnded := !combatHasActivePlayersLocked(combat) || !combatHasLivingNPCsLocked(combat)
		var endEvent *CombatEnded
		var respawnEvents []PlayerRespawned
		if combatEnded {
			endEvent, respawnEvents, err = w.endCombatLocked(combat)
			if err != nil {
				return nil, err
			}
		}

		if attackEvent != nil {
			events = append(events, *attackEvent)
		}
		if damageEvent != nil {
			events = append(events, *damageEvent)
		}
		if deathEvent != nil {
			events = append(events, *deathEvent)
		}
		if endEvent != nil {
			events = append(events, *endEvent)
			for _, respawn := range respawnEvents {
				events = append(events, respawn)
			}
			return events, nil
		}

		// Emit turn change if the turn moved to another participant after consuming the NPC action.
		if combat.TurnIndex >= 0 &&
			combat.TurnIndex < len(combat.Participants) &&
			combat.Participants[combat.TurnIndex].ID != previousParticipantID {
			if turnEvent := w.buildCombatTurnChangedLocked(combat); turnEvent != nil {
				events = append(events, *turnEvent)
			}
		}
	}
}

func (w *World) startCombatRoundLocked(combat *Combat) (*CombatTurnChanged, error) {
	if !combat.TurnsInitialized {
		return nil, fmt.Errorf("%w: combat turns have not been initialized", ErrInvalidWorldState)
	}

	if !combatHasLivingPlayersLocked(combat) {
		return nil, fmt.Errorf("%w: combat has no living players", ErrInvalidWorldState)
	}

	if !combatHasLivingNPCsLocked(combat) {
		return nil, fmt.Errorf("%w: all NPCs have been defeated", ErrInvalidWorldState)
	}

	if combatHasAvailableActionsLocked(combat) {
		return nil, fmt.Errorf("%w: combat still has available actions", ErrInvalidWorldState)
	}

	combat.Round++

	// Accumulate speed points (includes any player who joined mid-combat during the previous round).
	for i := range combat.Participants {
		participant := &combat.Participants[i]

		if !participant.Alive ||
			participant.Escaped ||
			(participant.IsPlayer && !participant.Connected) {
			continue
		}

		participant.SpeedPoints += calculateEffectiveSpeed(*participant)
	}

	combat.TurnIndex = -1
	highestSpeedPoints := -1

	for i, participant := range combat.Participants {
		if AvailableActions(participant) <= 0 {
			continue
		}

		if participant.SpeedPoints > highestSpeedPoints {
			highestSpeedPoints = participant.SpeedPoints
			combat.TurnIndex = i
		}
	}

	if combat.TurnIndex == -1 {
		return nil, fmt.Errorf("%w: no participants have available actions", ErrInvalidWorldState)
	}

	// Expire effects belonging to the participant whose turn begins this round.
	if _, err := w.updateCombatEffectsLocked(combat, combat.Participants[combat.TurnIndex].ID); err != nil {
		return nil, err
	}

	return w.buildCombatTurnChangedLocked(combat), nil
}

func (w *World) executeNPCCombatTurnLocked(
	combat *Combat,
) (*CombatAttack, *CombatDamage, *ParticipantDied, error) {
	if len(combat.Participants) == 0 ||
		combat.TurnIndex < 0 ||
		combat.TurnIndex >= len(combat.Participants) {
		return nil, nil, nil, ErrInvalidWorldState
	}

	attacker := &combat.Participants[combat.TurnIndex]

	if attacker.IsPlayer ||
		!attacker.Alive ||
		attacker.Escaped ||
		AvailableActions(*attacker) <= 0 {
		return nil, nil, nil, ErrInvalidCombatTurn
	}

	if _, exists := w.npcs[attacker.ID]; !exists {
		return nil, nil, nil, ErrNPCNotFound
	}

	target, err := selectNPCCombatTarget(combat)
	if err != nil {
		return nil, nil, nil, err
	}

	targetIndex := -1
	for i := range combat.Participants {
		if combat.Participants[i].ID == target.ID {
			targetIndex = i
			break
		}
	}
	if targetIndex == -1 {
		return nil, nil, nil, ErrInvalidCombatTarget
	}

	previousHP := target.Stats.HP
	previousAlive := target.Alive

	player, exists := w.players[target.ID]
	if !exists || player == nil {
		return nil, nil, nil, ErrPlayerNotFound
	}
	previousEntityHP := player.CombatStats.HP

	damage := calculateDamage(
		calculateEffectiveStat(*attacker, CombatStatAttack),
		calculateEffectiveStat(*target, CombatStatDefense),
	)

	damage, err = w.applyCombatDamageLocked(target, damage)
	if err != nil {
		return nil, nil, nil, err
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

	if err := w.consumeCombatActionLocked(combat); err != nil {
		combat.Participants[targetIndex].Stats.HP = previousHP
		combat.Participants[targetIndex].Alive = previousAlive
		w.players[target.ID].CombatStats.HP = previousEntityHP
		return nil, nil, nil, err
	}

	return attackEvent, damageEvent, deathEvent, nil
}

func (w *World) endCombatLocked(combat *Combat) (*CombatEnded, []PlayerRespawned, error) {
	defeatedNPCs, hasLivingNPC, err := w.collectCombatNPCResultsLocked(combat)
	if err != nil {
		return nil, nil, err
	}

	if hasLivingNPC && combatHasActivePlayersLocked(combat) {
		return nil, nil, fmt.Errorf("%w: combat is still ongoing", ErrInvalidWorldState)
	}

	victory := !hasLivingNPC

	if err := w.validateCombatEndLocked(combat, victory, defeatedNPCs); err != nil {
		return nil, nil, err
	}

	w.synchronizeCombatNPCsLocked(combat)
	respawnEvents := w.synchronizeCombatPlayersLocked(combat, victory)

	var rewards []CombatReward
	if victory {
		rewards, err = w.distributeCombatRewardsLocked(combat, defeatedNPCs)
		if err != nil {
			return nil, nil, err
		}
	}

	recipients := w.getCombatRecipientsLocked(combat)
	w.cleanupDisconnectedCombatPlayersLocked(combat)
	combat.Active = false

	return &CombatEnded{
		CombatID:   combat.ID,
		Rewards:    rewards,
		Recipients: recipients,
	}, respawnEvents, nil
}