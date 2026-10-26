package world

import (
	"fmt"
	"errors"
)

var ErrNoExit = errors.New("no exit in the requested direction")

// MovePlayer moves a player through an exit from their current room.
// The operation is atomic: the player's location is updated only if
// the player exists and the exit and destination are valid.
func (world *World) MovePlayer(playerName, direction string) error {
	if world == nil {
		return fmt.Errorf("world is nil")
	}

	if playerName == "" {
		return fmt.Errorf("player name cannot be empty")
	}

	if direction == "" {
		return fmt.Errorf("direction cannot be empty")
	}

	world.mu.Lock()
	defer world.mu.Unlock()

	// Find the player's current room.
	currentRoomID, exists := world.playerRooms[playerName]
	if !exists {
		return fmt.Errorf("player %q is not in the world", playerName)
	}

	// Verify that the current room exists.
	currentRoom, exists := world.rooms[currentRoomID]
	if !exists {
		return fmt.Errorf("current room %q does not exist", currentRoomID)
	}

	// Check whether the requested exit exists.
	destinationRoomID, exists := currentRoom.Exits[direction]
	if !exists {
		return fmt.Errorf(
			"%w: direction %q from room %q",
			ErrNoExit,
			direction,
			currentRoomID,
		)
	}

	// Verify that the destination room exists.
	if _, exists := world.rooms[destinationRoomID]; !exists {
		return fmt.Errorf(
			"destination room %q for direction %q does not exist",
			destinationRoomID,
			direction,
		)
	}

	// Update the player's location only after all validations succeed.
	world.playerRooms[playerName] = destinationRoomID

	return nil
}