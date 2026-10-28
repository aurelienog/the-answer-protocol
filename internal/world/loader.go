package world

import (
	"encoding/json"
	"fmt"
	"os"
)

// Load reads a world definition from a JSON file and validates it.
func Load(path string) (*World, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read world file %q: %w", path, err)
	}

	// Temporary structure used only to decode the JSON file.
	var definition struct {
		Rooms map[string]Room `json:"rooms"`
		Items map[string]Item `json:"items"`
	}

	if err := json.Unmarshal(data, &definition); err != nil {
		return nil, fmt.Errorf("decode world file %q: %w", path, err)
	}

	gameWorld := &World{
		rooms:             definition.Rooms,
		playerRooms:       make(map[string]string),
		items:             definition.Items,
		itemInstances:     make(map[string]ItemInstance),
		roomItemInstances: make(map[string][]string),
		playerInventories: make(map[string][]string),
	}

	if err := gameWorld.initializeItemInstances(); err != nil {
		return nil, fmt.Errorf("initialize item instances in world file %q: %w", path, err)
	}

	if err := gameWorld.Validate(); err != nil {
		return nil, fmt.Errorf("validate world file %q: %w", path, err)
	}

	return gameWorld, nil
}

// Validate checks that the world definition is internally consistent.
func (w *World) Validate() error {
	if w == nil {
		return fmt.Errorf("world is nil")
	}

	if len(w.rooms) == 0 {
		return fmt.Errorf("world has no rooms")
	}

	// Validate item definitions.
	for key, item := range w.items {
		if key == "" || item.ID == "" {
			return fmt.Errorf("item ID cannot be empty")
		}

		if item.ID != key {
			return fmt.Errorf(
				"item key %q does not match item ID %q",
				key,
				item.ID,
			)
		}

		if item.Name == "" {
			return fmt.Errorf("item %q has an empty name", key)
		}

		switch item.Type {
		case ItemTypeCommon, ItemTypeQuest, ItemTypeUnique:
		default:
			return fmt.Errorf(
				"item %q has invalid type %q",
				key,
				item.Type,
			)
		}
	}

	// Validate rooms, exits and item references.
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

// initializeItemInstances creates one instance for each item quantity
// declared in the world definition.
func (w *World) initializeItemInstances() error {
	for roomID, room := range w.rooms {
		for itemID, quantity := range room.Items {
			if quantity <= 0 {
				return fmt.Errorf(
					"room %q has invalid quantity %d for item %q",
					roomID,
					quantity,
					itemID,
				)
			}

			if _, exists := w.items[itemID]; !exists {
				return fmt.Errorf(
					"room %q references nonexistent item %q",
					roomID,
					itemID,
				)
			}

			for i := 0; i < quantity; i++ {
				instanceID := fmt.Sprintf(
					"instance:%s:%d",
					itemID,
					i+1,
				)

				if _, exists := w.itemInstances[instanceID]; exists {
					return fmt.Errorf(
						"duplicate item instance ID %q",
						instanceID,
					)
				}

				w.itemInstances[instanceID] = ItemInstance{
					ID:     instanceID,
					ItemID: itemID,
				}

				w.roomItemInstances[roomID] = append(
					w.roomItemInstances[roomID],
					instanceID,
				)
			}
		}
	}

	return nil
}