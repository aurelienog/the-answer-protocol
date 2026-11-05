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
		case ItemTypeCommon,
			ItemTypeQuest,
			ItemTypeUnique,
			ItemTypeEquipment,
			ItemTypeConsumable:
		default:
			return fmt.Errorf(
				"item %q has invalid type %q",
				key,
				item.Type,
			)
		}

		// Validate combat attributes.
		if item.Attack < 0 ||
			item.Defense < 0 ||
			item.Speed < 0 ||
			item.HP < 0 {
			return fmt.Errorf(
				"item %q has negative combat attributes",
				key,
			)
		}

		// Equipment and consumables must provide at least one attribute.
		if item.Type == ItemTypeEquipment ||
			item.Type == ItemTypeConsumable {
			if item.Attack == 0 &&
				item.Defense == 0 &&
				item.Speed == 0 &&
				item.HP == 0 {
				return fmt.Errorf(
					"item %q of type %q has no combat attributes",
					key,
					item.Type,
				)
			}
		}
	}

	return nil
}