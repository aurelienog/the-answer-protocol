package world

import "fmt"

// validateRooms checks that rooms, exits, and room references are consistent.
func (w *World) validateRooms() error {
	uniqueItemRooms := make(map[string]string)

	for key, room := range w.rooms {
		if room.ID == "" {
			return fmt.Errorf("room key %q has an empty ID", key)
		}

		if key != room.ID {
			return fmt.Errorf(
				"room key %q does not match room ID %q",
				key,
				room.ID,
			)
		}

		if room.Name == "" {
			return fmt.Errorf("room %q has an empty name", key)
		}

		// Validate NPC references.
		seenNPCs := make(map[string]struct{})

		for _, npcID := range room.NPCs {
			if npcID == "" {
				return fmt.Errorf(
					"room %q references an NPC with an empty ID",
					key,
				)
			}

			if _, exists := w.npcs[npcID]; !exists {
				return fmt.Errorf(
					"room %q references nonexistent NPC %q",
					key,
					npcID,
				)
			}

			if _, exists := seenNPCs[npcID]; exists {
				return fmt.Errorf(
					"room %q references NPC %q more than once",
					key,
					npcID,
				)
			}

			seenNPCs[npcID] = struct{}{}
		}

		// Validate exits.
		for direction, destinationID := range room.Exits {
			if _, exists := w.rooms[destinationID]; !exists {
				return fmt.Errorf(
					"room %q has exit %q to nonexistent room %q",
					key,
					direction,
					destinationID,
				)
			}
		}

		// Validate item references and quantities.
		for itemID, quantity := range room.Items {
			item, exists := w.items[itemID]
			if !exists {
				return fmt.Errorf(
					"room %q references nonexistent item %q",
					key,
					itemID,
				)
			}

			if quantity <= 0 {
				return fmt.Errorf(
					"room %q has invalid quantity %d for item %q",
					key,
					quantity,
					itemID,
				)
			}

			// Unique items may exist in only one room.
			if item.Type == ItemTypeUnique {
				if quantity > 1 {
					return fmt.Errorf(
						"unique item %q has quantity %d in room %q; maximum is 1",
						itemID,
						quantity,
						key,
					)
				}

				if previousRoom, exists := uniqueItemRooms[itemID]; exists {
					return fmt.Errorf(
						"unique item %q appears in both room %q and room %q",
						itemID,
						previousRoom,
						key,
					)
				}

				uniqueItemRooms[itemID] = key
			}
		}
	}

	return nil
}