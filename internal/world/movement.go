package world

import (
	"fmt"
	"strings"
)

// MovePlayer moves a player through an exit from their current room.
// It returns a PlayerMoved event containing the room occupants captured
// immediately before the move.
// The operation is atomic.
func (w *World) MovePlayer(
	playerName, direction string,
) (PlayerMoved, error) {
	if w == nil {
		return PlayerMoved{}, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	playerName = strings.TrimSpace(playerName)
	direction = strings.ToLower(strings.TrimSpace(direction))

	if playerName == "" {
		return PlayerMoved{}, fmt.Errorf(
			"%w: player name cannot be empty",
			ErrInvalidArgument,
		)
	}

	if direction == "" {
		return PlayerMoved{}, fmt.Errorf(
			"%w: direction cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Find the player's current room.
	player, exists := w.players[playerName]
	if !exists {
		return PlayerMoved{}, fmt.Errorf(
			"%w: %q",
			ErrPlayerNotFound,
			playerName,
		)
	}

	currentRoomID := player.RoomID

	// Verify that the current room exists.
	currentRoom, exists := w.rooms[currentRoomID]
	if !exists {
		return PlayerMoved{}, fmt.Errorf(
			"%w: player %q references nonexistent room %q",
			ErrInvalidWorldState,
			playerName,
			currentRoomID,
		)
	}

	// Check whether the requested exit exists.
	destinationRoomID, exists := currentRoom.Exits[direction]
	if !exists {
		return PlayerMoved{}, fmt.Errorf(
			"%w: direction %q from room %q",
			ErrNoExit,
			direction,
			currentRoomID,
		)
	}

	// Verify that the destination room exists.
	if _, exists := w.rooms[destinationRoomID]; !exists {
		return PlayerMoved{}, fmt.Errorf(
			"%w: destination room %q for direction %q does not exist",
			ErrInvalidWorldState,
			destinationRoomID,
			direction,
		)
	}

	// Capture both room populations before changing the player's location.
	previousRoomPlayers := w.getPlayersInRoomLocked(currentRoomID)
	newRoomPlayers := w.getPlayersInRoomLocked(destinationRoomID)

	// Update the player's location only after all validations succeed.
	player.RoomID = destinationRoomID

	return PlayerMoved{
		PlayerName:          playerName,
		PreviousRoomID:      currentRoomID,
		NewRoomID:           destinationRoomID,
		PreviousRoomPlayers: previousRoomPlayers,
		NewRoomPlayers:      newRoomPlayers,
	}, nil
}