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

	var gameWorld World

	if err := json.Unmarshal(data, &gameWorld); err != nil {
		return nil, fmt.Errorf("decode world file %q: %w", path, err)
	}

	if err := gameWorld.Validate(); err != nil {
		return nil, fmt.Errorf("validate world file %q: %w", path, err)
	}

	return &gameWorld, nil
}

// Validate checks that the world definition is internally consistent.
func (w *World) Validate() error {
	if w == nil {
		return fmt.Errorf("world is nil")
	}

	if len(w.Rooms) == 0 {
		return fmt.Errorf("world has no rooms")
	}

	for key, room := range w.Rooms {
		if key == "" {
			return fmt.Errorf("room has an empty key")
		}

		if room.ID != key {
			return fmt.Errorf(
				"room key %q does not match its ID %q",
				key,
				room.ID,
			)
		}

		if room.Name == "" {
			return fmt.Errorf("room %q has no name", key)
		}

		for direction, destination := range room.Exits {
			if direction == "" {
				return fmt.Errorf(
					"room %q has an exit with an empty direction",
					key,
				)
			}

			if _, exists := w.Rooms[destination]; !exists {
				return fmt.Errorf(
					"room %q has an exit %q to unknown room %q",
					key,
					direction,
					destination,
				)
			}
		}
	}

	return nil
}