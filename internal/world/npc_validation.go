package world

import (
	"fmt"
	"strings"
)

// validateNPCs checks that all NPC definitions are internally consistent.
func (w *World) validateNPCs() error {
	for key, npc := range w.npcs {
		if err := validateNPCIdentity(key, npc); err != nil {
			return err
		}

		if err := validateNPCCombatStats(key, npc.CombatStats); err != nil {
			return err
		}

		if err := w.validateNPCRewards(key, npc.Rewards); err != nil {
			return err
		}
	}

	return nil
}

// validateNPCIdentity checks the NPC ID and name.
func validateNPCIdentity(key string, npc NPC) error {
	if strings.TrimSpace(key) == "" || strings.TrimSpace(npc.ID) == "" {
		return fmt.Errorf("NPC ID cannot be empty")
	}

	if npc.ID != key {
		return fmt.Errorf(
			"NPC key %q does not match NPC ID %q",
			key,
			npc.ID,
		)
	}

	if strings.TrimSpace(npc.Name) == "" {
		return fmt.Errorf("NPC %q has an empty name", key)
	}

	return nil
}

// validateNPCCombatStats checks that NPC combat statistics are valid.
func validateNPCCombatStats(key string, stats CombatStats) error {
	if stats.HP < 0 {
		return fmt.Errorf("NPC %q has negative HP %d", key, stats.HP)
	}

	if stats.MaxHP <= 0 {
		return fmt.Errorf("NPC %q has invalid max HP %d", key, stats.MaxHP)
	}

	if stats.HP > stats.MaxHP {
		return fmt.Errorf(
			"NPC %q has HP %d greater than max HP %d",
			key,
			stats.HP,
			stats.MaxHP,
		)
	}

	if stats.Attack <= 0 {
		return fmt.Errorf(
			"NPC %q has invalid attack %d: must be greater than zero",
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

	if stats.Speed <= 0 {
		return fmt.Errorf(
			"NPC %q has invalid speed %d: must be greater than zero",
			key,
			stats.Speed,
		)
	}

	return nil
}

// validateNPCRewards checks that NPC reward references exist.
func (w *World) validateNPCRewards(key string, rewards []string) error {
	for _, rewardItemID := range rewards {
		if strings.TrimSpace(rewardItemID) == "" {
			return fmt.Errorf("NPC %q has an empty reward item ID", key)
		}

		if _, exists := w.items[rewardItemID]; !exists {
			return fmt.Errorf(
				"NPC %q references nonexistent reward item %q",
				key,
				rewardItemID,
			)
		}
	}

	return nil
}