package world

import (
	"fmt"
	"strings"
)

// StartCombat starts a combat against a hostile NPC.
// The initiating player and their group members in the same room
// become participants.
func (w *World) StartCombat(
    playerName string,
    npcID string,
) (*Combat, *CombatStarted, error) {
	if w == nil {
		return nil, nil, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	playerName = strings.TrimSpace(playerName)
	npcID = strings.TrimSpace(npcID)

	if playerName == "" || npcID == "" {
		return nil, nil, fmt.Errorf(
			"%w: player name and NPC ID cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	player, exists := w.players[playerName]
	if !exists {
		return nil, nil, fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
	}

	if player.CombatStats.HP <= 0 {
		return nil, nil, fmt.Errorf(
			"%w: player %q is defeated",
			ErrInvalidWorldState,
			playerName,
		)
	}

	room, exists := w.rooms[player.RoomID]
	if !exists {
		return nil, nil, fmt.Errorf(
			"%w: player %q references nonexistent room %q",
			ErrInvalidWorldState,
			playerName,
			player.RoomID,
		)
	}

	resolvedNPCID, err := w.resolveNPCIDInRoomLocked(room, npcID)
	if err != nil {
		return nil, nil, err
	}
	w.refreshNPCRespawnLocked(resolvedNPCID)

	npcID = resolvedNPCID
	npc := w.npcs[npcID]

	if !npc.Hostile {
		return nil, nil, fmt.Errorf("%w: %q", ErrNPCNotHostile, npcID)
	}

	if npc.CombatStats.HP <= 0 {
		return nil, nil, fmt.Errorf(
			"%w: %q",
			ErrNPCRespawning,
			npcID,
		)
	}

	// Prevent the initiating player from joining multiple combats.
	if w.playerInActiveCombatLocked(playerName) {
		return nil, nil, fmt.Errorf(
			"%w: %q",
			ErrPlayerAlreadyInCombat,
			playerName,
		)
	}

	// Prevent the target NPC from joining multiple combats.
	if w.npcInActiveCombatLocked(npcID) {
		return nil, nil, fmt.Errorf(
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

		if groupPlayer.RoomID != player.RoomID {
			return nil
		}

		if groupPlayer.CombatStats.HP <= 0 {
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

	// Add the player who initiated the combat first.
	if err := addPlayer(playerName); err != nil {
		return nil, nil, err
	}

	// Add the other members of the player's group.
	if groupID != "" {
		group, exists := w.groups[groupID]
		if !exists {
			return nil, nil, ErrGroupNotFound
		}

		for _, member := range group.Members {
			if member == playerName {
				continue
			}

			if err := addPlayer(member); err != nil {
				return nil, nil, err
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
		TurnIndex:    0,
		Round:        1,
		Active:       true,
	}

	w.combats[combatID] = combat

	event := &CombatStarted{
		CombatID:     combat.ID,
		RoomID:       combat.RoomID,
		Participants: make([]string, 0),
		Recipients:   w.getCombatRecipientsLocked(combat),
	}

	for _, participant := range combat.Participants {
		if participant.IsPlayer {
			event.Participants = append(event.Participants, participant.ID)
		}
	}

	return cloneCombat(combat), event, nil
}

// removePlayerFromCombatsLocked removes a player from active combats and
// advances the turn when necessary.
// The caller must hold w.mu.Lock().
func (w *World) removePlayerFromCombatsLocked(playerName string) {
	for _, combat := range w.combats {
		if combat == nil || !combat.Active {
			continue
		}

		for i, participant := range combat.Participants {
			if !participant.IsPlayer || participant.ID != playerName {
				continue
			}

			previousTurnIndex := combat.TurnIndex
			combat.Participants = append(
				combat.Participants[:i],
				combat.Participants[i+1:]...,
			)

			if !combatHasActivePlayersLocked(combat) {
				combat.TurnIndex = -1
			} else if previousTurnIndex > i {
				combat.TurnIndex = previousTurnIndex - 1
			} else if previousTurnIndex == i {
				combat.TurnIndex = -1
				for offset := 0; offset < len(combat.Participants); offset++ {
					index := (i + offset) % len(combat.Participants)
					if AvailableActions(combat.Participants[index]) > 0 {
						combat.TurnIndex = index
						break
					}
				}
			}

			break
		}
	}
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

// StartCombatRound starts the next combat round.
func (w *World) StartCombatRound(
	combatID string,
) (*Combat, *CombatTurnChanged, error) {
	if w == nil {
		return nil, nil, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	combatID = strings.TrimSpace(combatID)
	if combatID == "" {
		return nil, nil, ErrInvalidArgument
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	combat, exists := w.combats[combatID]
	if !exists || combat == nil || !combat.Active {
		return nil, nil, fmt.Errorf(
			"%w: active combat %q not found",
			ErrInvalidWorldState,
			combatID,
		)
	}

	if !combat.TurnsInitialized {
		return nil, nil, fmt.Errorf(
			"%w: combat turns have not been initialized",
			ErrInvalidWorldState,
		)
	}

	if !combatHasLivingPlayersLocked(combat) {
		return nil, nil, fmt.Errorf(
			"%w: combat has no living players",
			ErrInvalidWorldState,
		)
	}

	if !combatHasLivingNPCsLocked(combat) {
		return nil, nil, fmt.Errorf(
			"%w: all NPCs have been defeated",
			ErrInvalidWorldState,
		)
	}

	// A new round can only start after all available actions are exhausted.
	if combatHasAvailableActionsLocked(combat) {
		return nil, nil, fmt.Errorf(
			"%w: combat still has available actions",
			ErrInvalidWorldState,
		)
	}

	// Advance to the next round.
	combat.Round++

	// Accumulate speed using the effects currently active.
	for i := range combat.Participants {
		participant := &combat.Participants[i]

		if !participant.Alive ||
			participant.Escaped ||
			(participant.IsPlayer && !participant.Connected) {
			continue
		}

		participant.SpeedPoints += calculateEffectiveSpeed(*participant)
	}

	// Effects are not updated here.
	// Their duration is reduced when their owner starts their next turn.

	// Select the participant with the highest available speed points.
	// Ties preserve the participant order.
	combat.TurnIndex = -1
	highestSpeedPoints := -1

	for i := range combat.Participants {
		participant := combat.Participants[i]

		if AvailableActions(participant) <= 0 {
			continue
		}

		if participant.SpeedPoints > highestSpeedPoints {
			highestSpeedPoints = participant.SpeedPoints
			combat.TurnIndex = i
		}
	}
	if combat.TurnIndex == -1 {
		return nil, nil, fmt.Errorf(
			"%w: no participants have available actions",
			ErrInvalidWorldState,
		)
	}

	event := w.buildCombatTurnChangedLocked(combat)

	return cloneCombat(combat), event, nil

}

// IsCombatRoundComplete reports whether all active participants have exhausted their actions.
func (w *World) IsCombatRoundComplete(
	combatID string,
) (bool, error) {
	if w == nil {
		return false, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	combatID = strings.TrimSpace(combatID)
	if combatID == "" {
		return false, ErrInvalidArgument
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

	if !combat.TurnsInitialized {
		return false, fmt.Errorf(
			"%w: combat turns have not been initialized",
			ErrInvalidWorldState,
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

