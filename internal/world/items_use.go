package world

import "fmt"

// UseItem consumes an inventory item outside combat.
func (w *World) UseItem(playerName, itemID string) (ItemUsed, error) {
	if w == nil {
		return ItemUsed{}, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.players[playerName]; !exists {
		return ItemUsed{}, fmt.Errorf(
			"%w: %q",
			ErrPlayerNotFound,
			playerName,
		)
	}

	// Combat usage must go through COMBAT USE.
	if w.playerInActiveCombatLocked(playerName) {
		return ItemUsed{}, fmt.Errorf(
			"%w: player %q",
			ErrPlayerAlreadyInCombat,
			playerName,
		)
	}

	return w.useItemLocked(playerName, itemID)
}

// useItemLocked contains the shared consumable logic.
// The caller must hold w.mu.Lock().
func (w *World) useItemLocked(
	playerName string,
	itemID string,
) (ItemUsed, error) {
	inventory := w.playerInventories[playerName]

	instanceIndex := -1
	var selectedInstanceID string
	var item Item

	// Find the requested item instance.
	for i, instanceID := range inventory {
		instance, exists := w.itemInstances[instanceID]
		if !exists {
			return ItemUsed{}, fmt.Errorf(
				"%w: inventory references nonexistent instance %q",
				ErrInvalidWorldState,
				instanceID,
			)
		}

		definition, exists := w.items[instance.ItemID]
		if !exists {
			return ItemUsed{}, fmt.Errorf(
				"%w: instance %q references nonexistent item %q",
				ErrInvalidWorldState,
				instanceID,
				instance.ItemID,
			)
		}

		if instance.ItemID == itemID {
			instanceIndex = i
			selectedInstanceID = instanceID
			item = definition
			break
		}
	}

	if instanceIndex == -1 {
		return ItemUsed{}, fmt.Errorf(
			"%w: player %q does not have item %q",
			ErrItemNotInInventory,
			playerName,
			itemID,
		)
	}

	if item.Type != ItemTypeConsumable {
		return ItemUsed{}, fmt.Errorf(
			"%w: item %q has type %q",
			ErrItemNotConsumable,
			itemID,
			item.Type,
		)
	}

	if item.HP <= 0 {
		return ItemUsed{}, fmt.Errorf(
			"%w: item %q has no usable effect",
			ErrItemHasNoEffect,
			itemID,
		)
	}

	player := w.players[playerName]

	oldHP := player.CombatStats.HP
	maxHP := player.CombatStats.MaxHP

	newHP := oldHP + item.HP
	if newHP > maxHP {
		newHP = maxHP
	}

	healing := newHP - oldHP

	if healing <= 0 {
		return ItemUsed{}, fmt.Errorf(
			"%w: player %q already has maximum HP",
			ErrItemHasNoEffect,
			playerName,
		)
	}

	// Apply the healing effect.
	player.CombatStats.HP = newHP

	// Consume the inventory instance.
	inventory = append(
		inventory[:instanceIndex],
		inventory[instanceIndex+1:]...,
	)

	if len(inventory) == 0 {
		delete(w.playerInventories, playerName)
	} else {
		w.playerInventories[playerName] = inventory
	}

	delete(w.itemInstances, selectedInstanceID)

	return ItemUsed{
		PlayerName: playerName,
		ItemID:     itemID,
		StatChanges: map[string]int{
			CombatStatHP: healing,
		},
	}, nil
}
