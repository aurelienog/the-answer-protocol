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
		Rooms  map[string]Room  `json:"rooms"`
		Items  map[string]Item  `json:"items"`
		NPCs   map[string]NPC   `json:"npcs"`
		Quests map[string]Quest `json:"quests"`
	}

	if err := json.Unmarshal(data, &definition); err != nil {
		return nil, fmt.Errorf("decode world file %q: %w", path, err)
	}

	gameWorld := &World{
		rooms:              definition.Rooms,
		npcs:               definition.NPCs,
		quests:             definition.Quests,
		players:            make(map[string]*Player),
		items:              definition.Items,
		itemInstances:      make(map[string]ItemInstance),
		roomItemInstances:  make(map[string][]string),
		playerInventories:  make(map[string][]string),
		playerQuests:       make(map[string]map[string]PlayerQuest),
		groups:             make(map[string]Group),
		playerGroups:       make(map[string]string),
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

	if err := w.validateNPCs(); err != nil {
		return err
	}

	if err := w.validateItems(); err != nil {
		return err
	}

	if err := w.validateQuests(); err != nil {
		return err
	}

	if err := w.validateRooms(); err != nil {
		return err
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
