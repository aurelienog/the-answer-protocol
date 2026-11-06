package world

import (
	"fmt"
	"strings"
)

var validExitDirections = map[string]struct{}{
	"north": {},
	"south": {},
	"east":  {},
	"west":  {},
}

// validateRooms checks that all room definitions are consistent.
func (w *World) validateRooms() error {
	uniqueItemRooms := make(map[string]string)
	uniqueNPCRooms := make(map[string]string)

	for key, room := range w.rooms {
		if err := validateRoomIdentity(key, room); err != nil {
			return err
		}
		if err := w.validateRoomNPCs(key, room, uniqueNPCRooms); err != nil {
			return err
		}
		if err := w.validateRoomExits(key, room); err != nil {
			return err
		}
		if err := w.validateRoomItems(key, room, uniqueItemRooms); err != nil {
			return err
		}
	}

	return nil
}

// validateRoomIdentity checks the room ID and name.
func validateRoomIdentity(key string, room Room) error {
	if strings.TrimSpace(room.ID) == "" {
		return fmt.Errorf("room key %q has an empty ID", key)
	}
	if key != room.ID {
		return fmt.Errorf("room key %q does not match room ID %q", key, room.ID)
	}
	if strings.TrimSpace(room.Name) == "" {
		return fmt.Errorf("room %q has an empty name", key)
	}

	return nil
}

// validateRoomNPCs checks NPC references and ensures each NPC belongs to one room.
func (w *World) validateRoomNPCs(
	roomID string,
	room Room,
	uniqueNPCRooms map[string]string,
) error {
	seenNPCs := make(map[string]struct{})

	for _, npcID := range room.NPCs {
		if strings.TrimSpace(npcID) == "" {
			return fmt.Errorf("room %q contains an empty NPC ID", roomID)
		}

		if _, exists := w.npcs[npcID]; !exists {
			return fmt.Errorf("room %q references unknown NPC %q", roomID, npcID)
		}

		if _, exists := seenNPCs[npcID]; exists {
			return fmt.Errorf("room %q contains duplicate NPC %q", roomID, npcID)
		}
		seenNPCs[npcID] = struct{}{}

		if previousRoom, exists := uniqueNPCRooms[npcID]; exists {
			return fmt.Errorf(
				"NPC %q is assigned to multiple rooms: %q and %q",
				npcID,
				previousRoom,
				roomID,
			)
		}

		uniqueNPCRooms[npcID] = roomID
	}

	return nil
}

// validateRoomExits checks that exit directions and destinations are valid.
func (w *World) validateRoomExits(roomID string, room Room) error {
	for direction, destinationID := range room.Exits {
		if _, valid := validExitDirections[direction]; !valid {
			return fmt.Errorf(
				"room %q has invalid exit direction %q",
				roomID,
				direction,
			)
		}

		if strings.TrimSpace(destinationID) == "" {
			return fmt.Errorf(
				"room %q has an empty destination for exit %q",
				roomID,
				direction,
			)
		}

		if _, exists := w.rooms[destinationID]; !exists {
			return fmt.Errorf(
				"room %q exit %q references unknown destination room %q",
				roomID,
				direction,
				destinationID,
			)
		}
	}

	return nil
}

// validateRoomItems checks item references, quantities, and unique item placement.
func (w *World) validateRoomItems(
	roomID string,
	room Room,
	uniqueItemRooms map[string]string,
) error {
	for itemID, quantity := range room.Items {
		item, exists := w.items[itemID]
		if !exists {
			return fmt.Errorf("room %q references unknown item %q", roomID, itemID)
		}

		if quantity <= 0 {
			return fmt.Errorf(
				"room %q has invalid quantity %d for item %q",
				roomID,
				quantity,
				itemID,
			)
		}

		if item.Type != ItemTypeUnique {
			continue
		}

		if quantity > 1 {
			return fmt.Errorf(
				"room %q contains multiple instances of unique item %q",
				roomID,
				itemID,
			)
		}

		if previousRoom, exists := uniqueItemRooms[itemID]; exists {
			return fmt.Errorf(
				"unique item %q is assigned to multiple rooms: %q and %q",
				itemID,
				previousRoom,
				roomID,
			)
		}

		uniqueItemRooms[itemID] = roomID
	}

	return nil
}