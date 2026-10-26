package world

import (
	"fmt"
	"sort"
)

// AddPlayer places a player in a room.
// It fails if the player name is empty, the room does not exist,
// or the player is already registered.
func (w *World) AddPlayer(playerName, roomID string) error {
	if w == nil {
		return fmt.Errorf("world is nil")
	}

	if playerName == "" {
		return fmt.Errorf("player name cannot be empty")
	}

	if roomID == "" {
		return fmt.Errorf("room ID cannot be empty")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.rooms[roomID]; !exists {
		return fmt.Errorf("room %q does not exist", roomID)
	}

	if _, exists := w.playerRooms[playerName]; exists {
		return fmt.Errorf("player %q is already in the world", playerName)
	}

	if w.playerRooms == nil {
		w.playerRooms = make(map[string]string)
	}

	w.playerRooms[playerName] = roomID

	return nil
}

// RemovePlayer removes a player from the world.
// It is safe to call this method more than once.
func (w *World) RemovePlayer(playerName string) {
	if w == nil {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	delete(w.playerRooms, playerName)
}

// GetPlayerRoom returns the ID of the room occupied by a player.
func (w *World) GetPlayerRoom(playerName string) (string, error) {
	if w == nil {
		return "", fmt.Errorf("world is nil")
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	roomID, exists := w.playerRooms[playerName]
	if !exists {
		return "", fmt.Errorf("player %q is not in the world", playerName)
	}

	return roomID, nil
}

// GetPlayersInRoom returns the names of players in a room.
// The result is sorted alphabetically for deterministic output.
func (w *World) GetPlayersInRoom(roomID string) ([]string, error) {
	if w == nil {
		return nil, fmt.Errorf("world is nil")
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if _, exists := w.rooms[roomID]; !exists {
		return nil, fmt.Errorf("room %q does not exist", roomID)
	}

	players := make([]string, 0)

	for playerName, currentRoom := range w.playerRooms {
		if currentRoom == roomID {
			players = append(players, playerName)
		}
	}

	sort.Strings(players)

	return players, nil
}
