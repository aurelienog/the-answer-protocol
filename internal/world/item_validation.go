package world

import (
	"fmt"
	"strings"
)

// validateItems checks that all item definitions are internally consistent.
func (w *World) validateItems() error {
	for key, item := range w.items {
		if err := validateItemIdentity(key, item); err != nil {
			return err
		}

		if err := validateItemType(key, item.Type); err != nil {
			return err
		}

		if err := validateItemCombatStats(key, item); err != nil {
			return err
		}
	}

	return nil
}

// validateItemIdentity checks the item ID and name.
func validateItemIdentity(key string, item Item) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("item map contains an empty key")
	}

	if strings.TrimSpace(item.ID) == "" {
		return fmt.Errorf("item %q has an empty ID", key)
	}

	if item.ID != key {
		return fmt.Errorf(
			"item key %q does not match item ID %q",
			key,
			item.ID,
		)
	}

	if strings.TrimSpace(item.Name) == "" {
		return fmt.Errorf("item %q has an empty name", key)
	}

	return nil
}

// validateItemType checks that the item type is supported.
func validateItemType(key string, itemType ItemType) error {
	switch itemType {
	case ItemTypeCommon,
		ItemTypeQuest,
		ItemTypeUnique,
		ItemTypeEquipment,
		ItemTypeConsumable:
		return nil
	default:
		return fmt.Errorf(
			"item %q has invalid type %q",
			key,
			itemType,
		)
	}
}

// validateItemCombatStats checks item combat attributes.
func validateItemCombatStats(key string, item Item) error {
	if item.Attack < 0 ||
		item.Defense < 0 ||
		item.Speed < 0 ||
		item.HP < 0 {
		return fmt.Errorf(
			"item %q has negative combat attributes",
			key,
		)
	}

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

	return nil
}