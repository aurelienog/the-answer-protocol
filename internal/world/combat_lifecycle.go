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
) (*Combat, error) {
	if w == nil {
		return nil, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	playerName = strings.TrimSpace(playerName)
	npcID = strings.TrimSpace(npcID)

	if playerName == "" || npcID == "" {
		return nil, fmt.Errorf(
			"%w: player name and NPC ID cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	player, exists := w.players[playerName]
	if !exists {
		return nil, fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
	}

	if player.CombatStats.HP <= 0 {
		return nil, fmt.Errorf(
			"%w: player %q is defeated",
			ErrInvalidWorldState,
			playerName,
		)
	}

	room, exists := w.rooms[player.RoomID]
	if !exists {
		return nil, fmt.Errorf(
			"%w: player %q references nonexistent room %q",
			ErrInvalidWorldState,
			playerName,
			player.RoomID,
		)
	}

	npc, exists := w.npcs[npcID]
	if !exists {
		return nil, fmt.Errorf("%w: %q", ErrNPCNotFound, npcID)
	}

	if !npc.Hostile {
		return nil, fmt.Errorf("%w: %q", ErrNPCNotHostile, npcID)
	}

	if npc.CombatStats.HP <= 0 {
		return nil, fmt.Errorf(
			"%w: NPC %q is defeated",
			ErrInvalidWorldState,
			npcID,
		)
	}

	// The target NPC must belong to the player's current room.
	npcInRoom := false
	for _, roomNPCID := range room.NPCs {
		if roomNPCID == npcID {
			npcInRoom = true
			break
		}
	}

	if !npcInRoom {
		return nil, fmt.Errorf(
			"%w: NPC %q is not in room %q",
			ErrInvalidWorldState,
			npcID,
			player.RoomID,
		)
	}

	// Prevent the initiating player from joining multiple combats.
	if w.playerInActiveCombatLocked(playerName) {
		return nil, fmt.Errorf(
			"%w: player %q is already in an active combat",
			ErrInvalidWorldState,
			playerName,
		)
	}

	// Prevent the target NPC from joining multiple combats.
	if w.npcInActiveCombatLocked(npcID) {
		return nil, fmt.Errorf(
			"%w: NPC %q is already in an active combat",
			ErrInvalidWorldState,
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
					ErrInvalidWorldState,
					name,
				)
			}
			// A group member already fighting elsewhere is excluded.
			return nil
		}

		participants = append(participants, CombatParticipant{
			ID:        name,
			Name:      name,
			IsPlayer:  true,
			Connected: true,
			Stats:     groupPlayer.CombatStats,
			Alive:     true,
		})

		return nil
	}

	groupID := w.playerGroups[playerName]

	if groupID != "" {
		group, exists := w.groups[groupID]
		if !exists {
			return nil, fmt.Errorf(
				"%w: player %q references nonexistent group %q",
				ErrInvalidWorldState,
				playerName,
				groupID,
			)
		}

		for _, member := range group.Members {
			if err := addPlayer(member); err != nil {
				return nil, err
			}
		}
	} else {
		if err := addPlayer(playerName); err != nil {
			return nil, err
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

	return cloneCombat(combat), nil
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

            participant.Connected = false
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

	// Synchronize NPC HP.
	for _, participant := range combat.Participants {
		if participant.IsPlayer {
			continue
		}

		npc := w.npcs[participant.ID]
		npc.CombatStats.HP = participant.Stats.HP
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
			!participant.Escaped {
			return true
		}
	}

	return false
}