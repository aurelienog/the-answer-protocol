package world

import "fmt"

// validateItems checks that all item definitions are internally consistent.
func (w *World) validateItems() error {
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

	return nil
}
