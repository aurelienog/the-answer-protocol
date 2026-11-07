package world

import (
	"fmt"
	"sort"
	"strings"
)

type Player struct {
    Name        string      `json:"name"`
    RoomID      string      `json:"room_id"`
    CombatStats CombatStats `json:"combat_stats"`
}

// AddPlayer places a player in a room.
// It fails if the player name is empty, the room does not exist,
// or the player is already registered.
func (w *World) AddPlayer(playerName, roomID string) error {
	if w == nil {
		return fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	playerName = strings.TrimSpace(playerName)
	roomID = strings.TrimSpace(roomID)

	if playerName == "" {
		return fmt.Errorf("%w: player name cannot be empty", ErrInvalidArgument)
	}

	if roomID == "" {
		return fmt.Errorf("%w: room ID cannot be empty", ErrInvalidArgument)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.rooms[roomID]; !exists {
		return fmt.Errorf(
			"%w: room %q does not exist",
			ErrInvalidWorldState,
			roomID,
		)
	}

	if _, exists := w.players[playerName]; exists {
		return fmt.Errorf(
			"%w: player %q is already in the world",
			ErrPlayerAlreadyExists,
			playerName,
		)
	}

	if w.players == nil {
		w.players = make(map[string]*Player)
	}

	combatStats := CombatStats{
		HP:      DefaultPlayerHP,
		MaxHP:   DefaultPlayerHP,
		Attack:  DefaultPlayerAttack,
		Defense: DefaultPlayerDefense,
		Speed:   DefaultPlayerSpeed,
	}

	// Restore the player's combat state if they reconnect
	// while participating in an active combat.
	for _, combat := range w.combats {
		if combat == nil || !combat.Active {
			continue
		}

		for i := range combat.Participants {
			participant := &combat.Participants[i]

			if !participant.IsPlayer || participant.ID != playerName {
				continue
			}

			if participant.Escaped {
				continue
			}

			if combat.RoomID != roomID {
				return fmt.Errorf(
					"%w: player %q must reconnect in combat room %q",
					ErrInvalidWorldState,
					playerName,
					combat.RoomID,
				)
			}

			if participant.Connected {
				return fmt.Errorf(
					"%w: player %q is already connected to combat",
					ErrInvalidWorldState,
					playerName,
				)
			}

			combatStats = participant.Stats
			participant.Connected = true
			participant.SpeedPoints = 0
			break
		}
	}

	w.players[playerName] = &Player{
		Name:        playerName,
		RoomID:      roomID,
		CombatStats: combatStats,
	}

	return nil
}

// RemovePlayer returns the player's unique items to their current room
// and removes the player's state from the world.
// It also returns the IDs of active combats that must be ended.
// It is safe to call this method more than once.
func (w *World) RemovePlayer(
	playerName string,
) (PlayerLeft, UniqueItemsReturned, *CombatTurnChanged, []string, error) {
	if w == nil {
		return PlayerLeft{}, UniqueItemsReturned{}, nil, nil,
			fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	playerName = strings.TrimSpace(playerName)
	if playerName == "" {
		return PlayerLeft{}, UniqueItemsReturned{}, nil, nil,
			fmt.Errorf("%w: player name cannot be empty", ErrInvalidArgument)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	player, exists := w.players[playerName]
	if !exists {
		return PlayerLeft{}, UniqueItemsReturned{}, nil, nil,
			fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
	}

	roomID := player.RoomID

	if _, exists := w.rooms[roomID]; !exists {
		return PlayerLeft{}, UniqueItemsReturned{}, nil, nil,
			fmt.Errorf(
				"%w: player %q references nonexistent room %q",
				ErrInvalidWorldState,
				playerName,
				roomID,
			)
	}

	// Capture room occupants before removing the player.
	roomPlayers := w.getPlayersInRoomLocked(roomID)

	// Validate the inventory before modifying the world.
	inventory := w.playerInventories[playerName]
	uniqueInstanceIDs := make([]string, 0)

	for _, instanceID := range inventory {
		instance, exists := w.itemInstances[instanceID]
		if !exists {
			return PlayerLeft{}, UniqueItemsReturned{}, nil, nil,
				fmt.Errorf(
					"%w: inventory references nonexistent instance %q",
					ErrInvalidWorldState,
					instanceID,
				)
		}

		item, exists := w.items[instance.ItemID]
		if !exists {
			return PlayerLeft{}, UniqueItemsReturned{}, nil, nil,
				fmt.Errorf(
					"%w: instance %q references nonexistent item %q",
					ErrInvalidWorldState,
					instanceID,
					instance.ItemID,
				)
		}

		if item.Type == ItemTypeUnique {
			uniqueInstanceIDs = append(uniqueInstanceIDs, instanceID)
		}
	}

	// Remove the player from their group before disconnecting them.
	if err := w.removePlayerFromGroupLocked(playerName); err != nil {
		return PlayerLeft{}, UniqueItemsReturned{}, nil, nil, err
	}

	// Find the player's active combat before disconnecting them.
	combat := w.findActiveCombatForPlayerLocked(playerName)

	// Mark the player as disconnected from active combats.
	w.removePlayerFromCombatsLocked(playerName)

	// Identify combats that no longer have active players.
	combatsToEnd := make([]string, 0)

	if combat != nil && !combatHasActivePlayersLocked(combat) {
		combatsToEnd = append(combatsToEnd, combat.ID)
	}

	// Capture the new combat turn if the combat still has active players.
	var turnEvent *CombatTurnChanged
	if combat != nil && len(combatsToEnd) == 0 {
		turnEvent = w.buildCombatTurnChangedLocked(combat)
	}

	// Return the existing unique instances to the room.
	w.roomItemInstances[roomID] = append(
		w.roomItemInstances[roomID],
		uniqueInstanceIDs...,
	)

	// Remove returned unique items from the player's inventory.
	if len(uniqueInstanceIDs) > 0 {
		uniqueSet := make(map[string]bool, len(uniqueInstanceIDs))

		for _, instanceID := range uniqueInstanceIDs {
			uniqueSet[instanceID] = true
		}

		remainingInventory := make([]string, 0, len(inventory))

		for _, instanceID := range inventory {
			if !uniqueSet[instanceID] {
				remainingInventory = append(remainingInventory, instanceID)
			}
		}

		w.playerInventories[playerName] = remainingInventory
	}

	// Preserve the remaining inventory while the player remains in an active combat.
	if !w.playerInActiveCombatLocked(playerName) {
		w.deletePlayerInventoryLocked(playerName)
	}

	delete(w.players, playerName)

	leftEvent := PlayerLeft{
		PlayerName:  playerName,
		RoomID:      roomID,
		RoomPlayers: roomPlayers,
	}

	returnedEvent := UniqueItemsReturned{
		PlayerName: playerName,
		RoomID:     roomID,
		ItemIDs:    uniqueInstanceIDs,
	}

	return leftEvent, returnedEvent, turnEvent, combatsToEnd, nil
}

// GetPlayerRoom returns the ID of the room occupied by a player.
func (w *World) GetPlayerRoom(playerName string) (string, error) {
	if w == nil {
		return "", fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	player, exists := w.players[playerName]
	if !exists {
		return "", fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
	}

	roomID := player.RoomID

	if _, exists := w.rooms[roomID]; !exists {
		return "", fmt.Errorf(
			"%w: player %q references nonexistent room %q",
			ErrInvalidWorldState,
			playerName,
			roomID,
		)
	}

	return roomID, nil
}

// GetPlayersInRoom returns the names of players in a room.
// The result is sorted alphabetically for deterministic output.
func (w *World) GetPlayersInRoom(roomID string) ([]string, error) {
	if w == nil {
		return nil, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if _, exists := w.rooms[roomID]; !exists {
		return nil, fmt.Errorf(
			"%w: room %q does not exist",
			ErrInvalidWorldState,
			roomID,
		)
	}

	players := make([]string, 0)

	for playerName, player := range w.players {
		if player.RoomID == roomID {
			players = append(players, playerName)
		}
	}

	sort.Strings(players)

	return players, nil
}

// getPlayersInRoomLocked returns the players in a room.
// The caller must hold at least a read lock on the world.
func (w *World) getPlayersInRoomLocked(roomID string) []string {
	players := make([]string, 0)

	for playerName, player := range w.players {
		if player.RoomID == roomID {
			players = append(players, playerName)
		}
	}

	sort.Strings(players)

	return players
}

// GetPlayerCount returns the total number of players in the world.
func (w *World) GetPlayerCount() (int, error) {
	if w == nil {
		return 0, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	return len(w.players), nil
}

// GetPlayerCombatStats returns a copy of a player's combat statistics.
func (w *World) GetPlayerCombatStats(
	playerName string,
) (CombatStats, error) {
	if w == nil {
		return CombatStats{}, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	playerName = strings.TrimSpace(playerName)

	if playerName == "" {
		return CombatStats{}, fmt.Errorf(
			"%w: player name cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	player, exists := w.players[playerName]
	if !exists {
		return CombatStats{}, fmt.Errorf(
			"%w: %q",
			ErrPlayerNotFound,
			playerName,
		)
	}

	return player.CombatStats, nil
}

// GetActiveCombatForPlayer returns a copy of the player's active combat.
func (w *World) GetActiveCombatForPlayer(
    playerName string,
) (*Combat, error) {
    if w == nil {
        return nil, fmt.Errorf(
            "%w: world is nil",
            ErrInvalidWorldState,
        )
    }

    playerName = strings.TrimSpace(playerName)

    if playerName == "" {
        return nil, fmt.Errorf(
            "%w: player name cannot be empty",
            ErrInvalidArgument,
        )
    }

    w.mu.RLock()
    defer w.mu.RUnlock()

    combat := w.findActiveCombatForPlayerLocked(playerName)
    if combat == nil {
        return nil, ErrPlayerNotInActiveCombat
    }

    return cloneCombat(combat), nil
}

// GetPlayerReconnectRoom returns the room of an active combat
// in which the player can reconnect.
func (w *World) GetPlayerReconnectRoom(
    playerName string,
) (string, error) {
    if w == nil {
        return "", fmt.Errorf(
            "%w: world is nil",
            ErrInvalidWorldState,
        )
    }

    playerName = strings.TrimSpace(playerName)

    if playerName == "" {
        return "", fmt.Errorf(
            "%w: player name cannot be empty",
            ErrInvalidArgument,
        )
    }

    w.mu.RLock()
    defer w.mu.RUnlock()

    var reconnectRoom string

    for _, combat := range w.combats {
        if combat == nil || !combat.Active {
            continue
        }

        for _, participant := range combat.Participants {
            if !participant.IsPlayer ||
                participant.ID != playerName ||
                participant.Escaped ||
                participant.Connected {
                continue
            }

            if reconnectRoom != "" {
                return "", fmt.Errorf(
                    "%w: player %q belongs to multiple active combats",
                    ErrInvalidWorldState,
                    playerName,
                )
            }

            if _, exists := w.rooms[combat.RoomID]; !exists {
                return "", fmt.Errorf(
                    "%w: combat %q references nonexistent room %q",
                    ErrInvalidWorldState,
                    combat.ID,
                    combat.RoomID,
                )
            }

            reconnectRoom = combat.RoomID
        }
    }

    if reconnectRoom == "" {
        return "", fmt.Errorf(
            "%w: %q",
            ErrPlayerNotInActiveCombat,
            playerName,
        )
    }

    return reconnectRoom, nil
}
// deletePlayerInventoryLocked removes a player's inventory and its item instances.
// The caller must hold w.mu.Lock().
func (w *World) deletePlayerInventoryLocked(playerName string) {
	for _, instanceID := range w.playerInventories[playerName] {
		delete(w.itemInstances, instanceID)
	}

	delete(w.playerInventories, playerName)
}