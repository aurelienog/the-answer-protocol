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

		if npc.HP < 0 {
			return fmt.Errorf(
				"NPC %q has negative HP %d",
				key,
				npc.HP,
			)
		}

		if npc.MaxHP <= 0 {
			return fmt.Errorf(
				"NPC %q has invalid max HP %d",
				key,
				npc.MaxHP,
			)
		}

		if npc.HP > npc.MaxHP {
			return fmt.Errorf(
				"NPC %q has HP %d greater than max HP %d",
				key,
				npc.HP,
				npc.MaxHP,
			)
		}

		if npc.Damage < 0 {
			return fmt.Errorf(
				"NPC %q has negative damage %d",
				key,
				npc.Damage,
			)
		}
	}

	return nil
}