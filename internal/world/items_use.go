package world

import (
	"fmt"
	"strings"
)

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
	instanceIndex, instanceID, item, err :=
		w.findConsumableInstanceLocked(playerName, itemID)
	if err != nil {
		return ItemUsed{}, err
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
	if player == nil {
		return ItemUsed{}, ErrPlayerNotFound
	}

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

	player.CombatStats.HP = newHP

	w.consumeItemInstanceLocked(
		playerName,
		instanceIndex,
		instanceID,
	)

	return ItemUsed{
		PlayerName: playerName,
		ItemID:     itemID,
		StatChanges: map[string]int{
			CombatStatHP: healing,
		},
	}, nil
}

// CombatUseItem consumes an item during the player's combat turn.
// Using an item does not consume combat action points.
func (w *World) CombatUseItem(
	combatID string,
	playerName string,
	itemID string,
) (*Combat, ItemUsed, error) {
	if w == nil {
		return nil, ItemUsed{}, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	combatID = strings.TrimSpace(combatID)
	playerName = strings.TrimSpace(playerName)
	itemID = strings.TrimSpace(itemID)

	if combatID == "" || playerName == "" || itemID == "" {
		return nil, ItemUsed{}, ErrInvalidArgument
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	combat, err := w.getActiveCombatLocked(combatID)
	if err != nil {
		return nil, ItemUsed{}, err
	}

	participant, err := w.getCurrentAttackerLocked(
		combat,
		playerName,
	)
	if err != nil {
		return nil, ItemUsed{}, err
	}

	if !participant.IsPlayer {
		return nil, ItemUsed{}, ErrInvalidCombatTurn
	}

	instanceIndex, instanceID, item, err :=
		w.findConsumableInstanceLocked(playerName, itemID)
	if err != nil {
		return nil, ItemUsed{}, err
	}

	if item.Type != ItemTypeConsumable {
		return nil, ItemUsed{}, fmt.Errorf(
			"%w: item %q has type %q",
			ErrItemNotConsumable,
			itemID,
			item.Type,
		)
	}

	if item.HP <= 0 {
		return nil, ItemUsed{}, fmt.Errorf(
			"%w: item %q has no usable effect",
			ErrItemHasNoEffect,
			itemID,
		)
	}

	oldHP := participant.Stats.HP
	maxHP := participant.Stats.MaxHP

	newHP := oldHP + item.HP
	if newHP > maxHP {
		newHP = maxHP
	}

	healing := newHP - oldHP

	if healing <= 0 {
		return nil, ItemUsed{}, fmt.Errorf(
			"%w: player %q already has maximum HP",
			ErrItemHasNoEffect,
			playerName,
		)
	}

	if !participant.IsPlayer {
		return nil, ItemUsed{}, ErrInvalidCombatTurn
	}

	player := w.players[playerName]
	if player == nil {
		return nil, ItemUsed{}, ErrPlayerNotFound
	}

	// Apply healing to the combat participant.
	participant.Stats.HP = newHP

	// Keep the player's persistent HP synchronized.
	player.CombatStats.HP = newHP

	// Consume the item.
	w.consumeItemInstanceLocked(
		playerName,
		instanceIndex,
		instanceID,
	)

	event := ItemUsed{
		PlayerName: playerName,
		ItemID:     itemID,
		StatChanges: map[string]int{
			CombatStatHP: healing,
		},
	}

	// No action points are consumed and the turn does not advance.
	return cloneCombat(combat), event, nil
}


// findConsumableInstanceLocked locates an item instance in the player's inventory.
// The caller must hold w.mu.Lock().
func (w *World) findConsumableInstanceLocked(
	playerName string,
	itemID string,
) (int, string, Item, error) {
	inventory := w.playerInventories[playerName]

	for i, instanceID := range inventory {
		instance, exists := w.itemInstances[instanceID]
		if !exists {
			return -1, "", Item{}, fmt.Errorf(
				"%w: inventory references nonexistent instance %q",
				ErrInvalidWorldState,
				instanceID,
			)
		}

		item, exists := w.items[instance.ItemID]
		if !exists {
			return -1, "", Item{}, fmt.Errorf(
				"%w: instance %q references nonexistent item %q",
				ErrInvalidWorldState,
				instanceID,
				instance.ItemID,
			)
		}

		if instance.ItemID == itemID {
			return i, instanceID, item, nil
		}
	}

	return -1, "", Item{}, fmt.Errorf(
		"%w: player %q does not have item %q",
		ErrItemNotInInventory,
		playerName,
		itemID,
	)
}

// consumeItemInstanceLocked removes an item instance from the inventory.
// The caller must hold w.mu.Lock().
func (w *World) consumeItemInstanceLocked(
	playerName string,
	instanceIndex int,
	instanceID string,
) {
	inventory := w.playerInventories[playerName]

	inventory = append(
		inventory[:instanceIndex],
		inventory[instanceIndex+1:]...,
	)

	if len(inventory) == 0 {
		delete(w.playerInventories, playerName)
	} else {
		w.playerInventories[playerName] = inventory
	}

	delete(w.itemInstances, instanceID)
}