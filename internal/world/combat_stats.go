package world

import "fmt"

// calculatePlayerCombatStatsLocked calculates a player's effective combat
// stats by adding the bonuses of all equipment and unique items in their inventory.
// The caller must hold w.mu.
func (w *World) calculatePlayerCombatStatsLocked(
	playerName string,
) (CombatStats, error) {
	player, exists := w.players[playerName]
	if !exists {
		return CombatStats{}, fmt.Errorf(
			"%w: %q",
			ErrPlayerNotFound,
			playerName,
		)
	}

	stats := player.CombatStats

	instanceIDs := w.playerInventories[playerName]

	for _, instanceID := range instanceIDs {
		instance, exists := w.itemInstances[instanceID]
		if !exists {
			return CombatStats{}, fmt.Errorf(
				"%w: item instance %q not found in player inventory",
				ErrInvalidWorldState,
				instanceID,
			)
		}

		item, exists := w.items[instance.ItemID]
		if !exists {
			return CombatStats{}, fmt.Errorf(
				"%w: item definition %q not found",
				ErrInvalidWorldState,
				instance.ItemID,
			)
		}

		if item.Type != ItemTypeEquipment && item.Type != ItemTypeUnique {
			continue
		}

		stats.Attack += item.Attack
		stats.Defense += item.Defense
		stats.Speed += item.Speed
	}

	return stats, nil
}