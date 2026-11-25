package world

import (
	"fmt"
	"strings"
)

const initialCombatAttackDamage = 10

// StartCombat starts a combat against a hostile NPC, initializes turn order,
// and resolves any initial NPC turns if the NPC acts first.
func (w *World) StartCombat(
	playerName string,
	npcID string,
) (*Combat, []any, error) {
	combat, _, events, err := w.startCombat(playerName, npcID, 0)
	return combat, events, err
}

// StartCombatWithInitialAttack starts combat and applies the attacker's opening
// hit before resolving any NPC turns.
func (w *World) StartCombatWithInitialAttack(
	playerName string,
	npcID string,
) (*Combat, int, []any, error) {
	return w.startCombat(playerName, npcID, initialCombatAttackDamage)
}

func (w *World) startCombat(
	playerName string,
	npcID string,
	initialDamage int,
) (*Combat, int, []any, error) {
	if w == nil {
		return nil, 0, nil, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	playerName = strings.TrimSpace(playerName)
	npcID = strings.TrimSpace(npcID)

	if playerName == "" || npcID == "" {
		return nil, 0, nil, fmt.Errorf(
			"%w: player name and NPC ID cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	player, exists := w.players[playerName]
	if !exists {
		return nil, 0, nil, fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
	}

	if player.CombatStats.HP <= 0 {
		return nil, 0, nil, fmt.Errorf(
			"%w: player %q is defeated",
			ErrInvalidWorldState,
			playerName,
		)
	}

	room, exists := w.rooms[player.RoomID]
	if !exists {
		return nil, 0, nil, fmt.Errorf(
			"%w: player %q references nonexistent room %q",
			ErrInvalidWorldState,
			playerName,
			player.RoomID,
		)
	}

	resolvedNPCID, err := w.resolveNPCIDInRoomLocked(room, npcID)
	if err != nil {
		return nil, 0, nil, err
	}
	respawnEvent, respawned := w.refreshNPCRespawnLocked(resolvedNPCID)

	npcID = resolvedNPCID
	npc := w.npcs[npcID]

	if !npc.Hostile {
		return nil, 0, nil, fmt.Errorf("%w: %q", ErrNPCNotHostile, npcID)
	}

	if npc.CombatStats.HP <= 0 {
		return nil, 0, nil, fmt.Errorf(
			"%w: %q",
			ErrNPCRespawning,
			npcID,
		)
	}

	if w.playerInActiveCombatLocked(playerName) {
		return nil, 0, nil, fmt.Errorf(
			"%w: %q",
			ErrPlayerAlreadyInCombat,
			playerName,
		)
	}

	if w.npcInActiveCombatLocked(npcID) {
		return nil, 0, nil, fmt.Errorf(
			"%w: %q",
			ErrNPCAlreadyInCombat,
			npcID,
		)
	}

	participants := make([]CombatParticipant, 0)

	addPlayer := func(name string) error {
		groupPlayer, exists := w.players[name]
		if !exists {
			return fmt.Errorf(
				"%w: group references nonexistent player %q",
				ErrInvalidWorldState,
				name,
			)
		}

		if groupPlayer.RoomID != player.RoomID || groupPlayer.CombatStats.HP <= 0 {
			return nil
		}

		if w.playerInActiveCombatLocked(name) {
			if name == playerName {
				return fmt.Errorf(
					"%w: player %q is already in an active combat",
					ErrPlayerAlreadyInCombat,
					name,
				)
			}
			return nil
		}

		effectiveStats, err := w.calculatePlayerCombatStatsLocked(name)
		if err != nil {
			return err
		}

		participants = append(participants, CombatParticipant{
			ID:        name,
			Name:      name,
			IsPlayer:  true,
			Connected: true,
			Stats:     effectiveStats,
			Alive:     true,
		})

		return nil
	}

	groupID := w.playerGroups[playerName]

	if err := addPlayer(playerName); err != nil {
		return nil, 0, nil, err
	}

	if groupID != "" {
		group, exists := w.groups[groupID]
		if !exists {
			return nil, 0, nil, ErrGroupNotFound
		}

		for _, member := range group.Members {
			if member == playerName {
				continue
			}

			if err := addPlayer(member); err != nil {
				return nil, 0, nil, err
			}
		}
	}

	participants = append(participants, CombatParticipant{
		ID:       npc.ID,
		Name:     npc.Name,
		IsPlayer: false,
		Stats:    npc.CombatStats,
		Alive:    true,
	})

	if w.combats == nil {
		w.combats = make(map[string]*Combat)
	}

	var combatID string
	for {
		w.nextCombatID++
		combatID = fmt.Sprintf("combat:%d", w.nextCombatID)

		if _, exists := w.combats[combatID]; !exists {
			break
		}
	}

	combat := &Combat{
		ID:           combatID,
		RoomID:       player.RoomID,
		GroupID:      groupID,
		Participants: participants,
		TurnIndex:    -1,
		Round:        1,
		Active:       true,
	}

	// Initialize first round turns before registering the combat in w.combats
	// so failed initialization requires zero cleanup.
	turnEvent, err := w.initializeCombatTurnsLocked(combat)
	if err != nil {
		return nil, 0, nil, err
	}

	w.combats[combatID] = combat

	startedEvent := CombatStarted{
		CombatID:     combat.ID,
		RoomID:       combat.RoomID,
		Participants: make([]CombatParticipantInfo, 0, len(combat.Participants)),
		Recipients:   w.getCombatRecipientsLocked(combat),
	}

	for _, participant := range combat.Participants {
		startedEvent.Participants = append(
			startedEvent.Participants,
			CombatParticipantInfo{
				ID:       participant.ID,
				Name:     participant.Name,
				IsPlayer: participant.IsPlayer,
				HP:       participant.Stats.HP,
				MaxHP:    participant.Stats.MaxHP,
				Alive:    participant.Alive,
			},
		)
	}

	events := make([]any, 0, 8)
	damage := 0
	if respawned {
		events = append(events, *respawnEvent)
	}
	events = append(events, startedEvent)

	if initialDamage > 0 {
		target := &combat.Participants[len(combat.Participants)-1]
		damage, err = w.applyCombatDamageLocked(target, initialDamage)
		if err != nil {
			return nil, 0, nil, err
		}

		events = append(events,
			CombatAttack{
				CombatID:     combat.ID,
				AttackerID:   playerName,
				AttackerName: player.Name,
				TargetID:     target.ID,
				TargetName:   target.Name,
				IsPlayer:     true,
				Recipients:   startedEvent.Recipients,
			},
			CombatDamage{
				CombatID:     combat.ID,
				AttackerID:   playerName,
				AttackerName: player.Name,
				TargetID:     target.ID,
				TargetName:   target.Name,
				Damage:       damage,
				RemainingHP:  target.Stats.HP,
				Recipients:   startedEvent.Recipients,
			},
		)

		if !target.Alive {
			events = append(events, ParticipantDied{
				CombatID:        combat.ID,
				ParticipantID:   target.ID,
				ParticipantName: target.Name,
				IsPlayer:        false,
				KillerID:        playerName,
				Recipients:      startedEvent.Recipients,
			})
		}
	}

	if turnEvent != nil && combatHasLivingNPCsLocked(combat) {
		events = append(events, *turnEvent)
	}

	// If the NPC won initiative, resolve its turn(s) immediately until a player's turn arrives.
	progressionEvents, err := w.resolveCombatProgressionLocked(combat)
	if err != nil {
		return nil, 0, nil, err
	}
	events = append(events, progressionEvents...)

	return cloneCombat(combat), damage, events, nil
}

// removePlayerFromCombatsLocked removes a player from active combats,
// advances turns, executes any resulting NPC actions, or ends combats
// that no longer have active players.
// The caller must hold w.mu.Lock().
func (w *World) removePlayerFromCombatsLocked(playerName string) ([]any, error) {
	events := make([]any, 0)

	for _, combat := range w.combats {
		if combat == nil || !combat.Active {
			continue
		}

		removed := false
		wasCurrentTurn := false

		for i, participant := range combat.Participants {
			if !participant.IsPlayer || participant.ID != playerName {
				continue
			}

			removed = true
			previousTurnIndex := combat.TurnIndex
			wasCurrentTurn = previousTurnIndex == i

			combat.Participants = append(
				combat.Participants[:i],
				combat.Participants[i+1:]...,
			)

			if !combatHasActivePlayersLocked(combat) || len(combat.Participants) == 0 {
				combat.TurnIndex = -1
			} else if previousTurnIndex > i {
				combat.TurnIndex = previousTurnIndex - 1
			} else if previousTurnIndex == i {
				combat.TurnIndex = -1
				for offset := 0; offset < len(combat.Participants); offset++ {
					index := (i + offset) % len(combat.Participants)
					candidate := combat.Participants[index]
					if !candidate.Alive ||
						candidate.Escaped ||
						(candidate.IsPlayer && !candidate.Connected) ||
						AvailableActions(candidate) <= 0 {
						continue
					}
					if _, err := w.updateCombatEffectsLocked(combat, candidate.ID); err != nil {
						return nil, err
					}
					combat.TurnIndex = index
					break
				}
			}

			break
		}

		if !removed {
			continue
		}

		// If the disconnected player held the active turn and someone else still has actions,
		// emit the immediate turn change before resolving subsequent NPC turns.
		if wasCurrentTurn &&
			combatHasActivePlayersLocked(combat) &&
			combat.TurnIndex >= 0 &&
			combat.TurnIndex < len(combat.Participants) {
			if turnEvent := w.buildCombatTurnChangedLocked(combat); turnEvent != nil {
				events = append(events, *turnEvent)
			}
		}

		// Atomically end the combat if no active players remain, or advance rounds / NPC turns.
		progressionEvents, err := w.resolveCombatProgressionLocked(combat)
		if err != nil {
			return nil, err
		}
		events = append(events, progressionEvents...)
	}

	return events, nil
}

func combatHasActivePlayersLocked(combat *Combat) bool {
	for _, participant := range combat.Participants {
		if participant.IsPlayer &&
			participant.Alive &&
			participant.Connected &&
			!participant.Escaped {
			return true
		}
	}

	return false
}

// GetCombat returns a copy of a combat by ID.
func (w *World) GetCombat(combatID string) (*Combat, error) {
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

	w.mu.RLock()
	defer w.mu.RUnlock()

	combat, exists := w.combats[combatID]
	if !exists || combat == nil {
		return nil, fmt.Errorf(
			"%w: combat %q not found",
			ErrInvalidWorldState,
			combatID,
		)
	}

	return cloneCombat(combat), nil
}
