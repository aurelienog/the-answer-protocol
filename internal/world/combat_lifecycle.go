package world

import (
	"fmt"
	"strings"
	"time"
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

// removePlayerFromCombatsLocked marks a player as disconnected
// without removing them from active combats.
// The caller must hold w.mu.Lock().
func (w *World) removePlayerFromCombatsLocked(playerName string) {
	for _, combat := range w.combats {
		if combat == nil || !combat.Active {
			continue
		}

		for i := range combat.Participants {
			participant := &combat.Participants[i]

			if !participant.IsPlayer || participant.ID != playerName {
				continue
			}

			wasCurrentTurn := combat.TurnIndex == i

			participant.Connected = false
			participant.SpeedPoints = 0

			// End the combat if no living connected players remain.
			if !combatHasActivePlayersLocked(combat) {
				combat.Active = false
				combat.TurnIndex = -1
				break
			}

			if wasCurrentTurn {
				foundNextTurn := false

				for offset := 1; offset < len(combat.Participants); offset++ {
					index := (i + offset) % len(combat.Participants)

					if AvailableActions(combat.Participants[index]) > 0 {
						combat.TurnIndex = index
						foundNextTurn = true
						break
					}
				}

				if !foundNextTurn {
					combat.TurnIndex = -1
				}
			}

			break
		}
	}
}

// EndCombat finishes a combat, synchronizes final HP
// and distributes rewards if the players win.
func (w *World) EndCombat(combatID string) (*Combat, error) {
	if w == nil {
		return nil, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
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

	// Collect NPC participants and determine whether any remain alive.
	defeatedNPCs := make([]NPC, 0)
	hasNPC := false
	hasLivingNPC := false

	for _, participant := range combat.Participants {
		if participant.IsPlayer {
			continue
		}

		hasNPC = true

		npc, exists := w.npcs[participant.ID]
		if !exists {
			return nil, fmt.Errorf(
				"%w: NPC %q not found",
				ErrInvalidWorldState,
				participant.ID,
			)
		}

		if participant.Alive && participant.Stats.HP > 0 {
			hasLivingNPC = true
		} else {
			defeatedNPCs = append(defeatedNPCs, npc)
		}
	}

	if !hasNPC {
		return nil, fmt.Errorf(
			"%w: combat has no NPC participants",
			ErrInvalidWorldState,
		)
	}

	// Determine the combat outcome.
	playersActive := combatHasActivePlayersLocked(combat)

	if hasLivingNPC && playersActive {
		return nil, fmt.Errorf(
			"%w: combat is still ongoing",
			ErrInvalidWorldState,
		)
	}

	victory := !hasLivingNPC

	// Validate connected players before modifying their state.
	for _, participant := range combat.Participants {
		if !participant.IsPlayer || !participant.Connected {
			continue
		}

		if _, exists := w.players[participant.ID]; !exists {
			return nil, fmt.Errorf(
				"%w: player %q not found",
				ErrInvalidWorldState,
				participant.ID,
			)
		}
	}

	// Validate rewards before modifying the world.
	if victory {
		if err := w.validateCombatRewardsLocked(combat, defeatedNPCs); err != nil {
			return nil, err
		}
	} else {
		// Validate the respawn room before modifying the world.
		if _, exists := w.rooms[InitialRoomID]; !exists {
			return nil, fmt.Errorf(
				"%w: initial room %q not found",
				ErrInvalidWorldState,
				InitialRoomID,
			)
		}
	}

	// Synchronize NPC HP and schedule respawn when defeated.
	for _, participant := range combat.Participants {
		if participant.IsPlayer {
			continue
		}

		npc := w.npcs[participant.ID]
		npc.CombatStats.HP = participant.Stats.HP

		if npc.CombatStats.HP <= 0 {
			respawnSeconds := npc.RespawnSeconds
			if respawnSeconds <= 0 {
				respawnSeconds = DefaultNPCRespawnSeconds
			}

			npc.RespawnAt = time.Now().Add(
				time.Duration(respawnSeconds) * time.Second,
			).Unix()
		}

		w.npcs[participant.ID] = npc
	}

	// Synchronize connected players.
	for _, participant := range combat.Participants {
		if !participant.IsPlayer || !participant.Connected {
			continue
		}

		player := w.players[participant.ID]

		if victory || participant.Escaped {
			player.CombatStats.HP = participant.Stats.HP
			continue
		}

		// Defeat: respawn connected players who did not escape.
		player.RoomID = InitialRoomID
		player.CombatStats.HP = player.CombatStats.MaxHP / 4
	}

	if victory {
		// Distribute rewards to every participant who did not escape.
		if err := w.distributeCombatRewardsLocked(combat, defeatedNPCs); err != nil {
			return nil, err
		}
	} else {
		// Defeat: discard inventories of disconnected participants.
		for _, participant := range combat.Participants {
			if !participant.IsPlayer ||
				participant.Connected ||
				participant.Escaped {
				continue
			}

			delete(w.playerInventories, participant.ID)
			delete(w.players, participant.ID)
		}
	}

	combat.Active = false

	return cloneCombat(combat), nil
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

