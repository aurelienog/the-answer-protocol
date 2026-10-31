package world

import "fmt"

// validateNPCs checks that all NPC definitions are internally consistent.
func (w *World) validateNPCs() error {
	for key, npc := range w.npcs {
		if key == "" || npc.ID == "" {
			return fmt.Errorf("NPC ID cannot be empty")
		}

		if npc.ID != key {
			return fmt.Errorf(
				"NPC key %q does not match NPC ID %q",
				key,
				npc.ID,
			)
		}

		if npc.Name == "" {
			return fmt.Errorf("NPC %q has an empty name", key)
		}

		stats := npc.CombatStats

		if stats.HP < 0 {
			return fmt.Errorf(
				"NPC %q has negative HP %d",
				key,
				stats.HP,
			)
		}

		if stats.MaxHP <= 0 {
			return fmt.Errorf(
				"NPC %q has invalid max HP %d",
				key,
				stats.MaxHP,
			)
		}

		if stats.HP > stats.MaxHP {
			return fmt.Errorf(
				"NPC %q has HP %d greater than max HP %d",
				key,
				stats.HP,
				stats.MaxHP,
			)
		}

		if stats.Attack < 0 {
			return fmt.Errorf(
				"NPC %q has negative attack %d",
				key,
				stats.Attack,
			)
		}

		if stats.Defense < 0 {
			return fmt.Errorf(
				"NPC %q has negative defense %d",
				key,
				stats.Defense,
			)
		}

		if stats.Speed < 0 {
			return fmt.Errorf(
				"NPC %q has negative speed %d",
				key,
				stats.Speed,
			)
		}

		// Validate NPC reward item references.
		for _, rewardItemID := range npc.Rewards {
			if _, exists := w.items[rewardItemID]; !exists {
				return fmt.Errorf(
					"NPC %q references nonexistent reward item %q",
					key,
					rewardItemID,
				)
			}
		}
	}

	return nil
}