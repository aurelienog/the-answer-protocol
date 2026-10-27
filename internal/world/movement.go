package world

import (
	"errors"
	"fmt"
)

var ErrNoExit = errors.New("no exit in the requested direction")

// MovePlayer moves a player through an exit from their current room.
// It returns a PlayerMoved event containing the room occupants captured
// immediately before the move.
// The operation is atomic.
func (world *World) MovePlayer(
	playerName, direction string,
) (PlayerMoved, error) {
	if world == nil {
		return PlayerMoved{}, fmt.Errorf("world is nil")
	}

	if playerName == "" {
		return PlayerMoved{}, fmt.Errorf("player name cannot be empty")
	}

	if direction == "" {
		return PlayerMoved{}, fmt.Errorf("direction cannot be empty")
	}

	world.mu.Lock()
	defer world.mu.Unlock()

	// Find the player's current room.
	currentRoomID, exists := world.playerRooms[playerName]
	if !exists {
		return PlayerMoved{}, fmt.Errorf(
			"player %q is not in the world",
			playerName,
		)
	}

	// Verify that the current room exists.
	currentRoom, exists := world.rooms[currentRoomID]
	if !exists {
		return PlayerMoved{}, fmt.Errorf(
			"current room %q does not exist",
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
	if _, exists := world.rooms[destinationRoomID]; !exists {
		return PlayerMoved{}, fmt.Errorf(
			"destination room %q for direction %q does not exist",
			destinationRoomID,
			direction,
		)
	}

	// Capture both room populations before changing the player's location.
	previousRoomPlayers := world.getPlayersInRoomLocked(currentRoomID)
	newRoomPlayers := world.getPlayersInRoomLocked(destinationRoomID)

	// Update the player's location only after all validations succeed.
	world.playerRooms[playerName] = destinationRoomID

	return PlayerMoved{
		PlayerName:          playerName,
		PreviousRoomID:      currentRoomID,
		NewRoomID:           destinationRoomID,
		PreviousRoomPlayers: previousRoomPlayers,
		NewRoomPlayers:      newRoomPlayers,
	}, nil
}
