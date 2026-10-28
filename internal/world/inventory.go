package world

import "fmt"

// TakeItem moves one concrete item instance from the player's current room
// to their inventory.
func (w *World) TakeItem(playerName, itemID string) error {
	if w == nil {
		return fmt.Errorf("world is nil")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	roomID, exists := w.playerRooms[playerName]
	if !exists {
		return fmt.Errorf("player %q is not in the world", playerName)
	}

	if _, exists := w.items[itemID]; !exists {
		return fmt.Errorf("item %q does not exist", itemID)
	}

	// Prevent the player from acquiring the same item definition twice.
	for _, instanceID := range w.playerInventories[playerName] {
		instance, exists := w.itemInstances[instanceID]
		if !exists {
			return fmt.Errorf(
				"inventory references nonexistent instance %q",
				instanceID,
			)
		}

		if instance.ItemID == itemID {
			return fmt.Errorf(
				"player %q already has item %q",
				playerName,
				itemID,
			)
		}
	}

	// Find one concrete instance of the requested item in the room.
	roomInstances := w.roomItemInstances[roomID]
	instanceIndex := -1
	var selectedInstanceID string

	for i, instanceID := range roomInstances {
		instance, exists := w.itemInstances[instanceID]
		if !exists {
			return fmt.Errorf(
				"room %q references nonexistent instance %q",
				roomID,
				instanceID,
			)
		}

		if instance.ItemID == itemID {
			instanceIndex = i
			selectedInstanceID = instanceID
			break
		}
	}

	if instanceIndex == -1 {
		return fmt.Errorf("item %q is not in room %q", itemID, roomID)
	}

	// Remove the instance from the room.
	w.roomItemInstances[roomID] = append(
		roomInstances[:instanceIndex],
		roomInstances[instanceIndex+1:]...,
	)

	// Add the same instance to the player's inventory.
	w.playerInventories[playerName] = append(
		w.playerInventories[playerName],
		selectedInstanceID,
	)

	return nil
}

// DropItem moves one common item instance from the player's inventory
// to their current room.
func (w *World) DropItem(playerName, itemID string) error {
	if w == nil {
		return fmt.Errorf("world is nil")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	roomID, exists := w.playerRooms[playerName]
	if !exists {
		return fmt.Errorf("player %q is not in the world", playerName)
	}

	if _, exists := w.rooms[roomID]; !exists {
		return fmt.Errorf("room %q does not exist", roomID)
	}

	if _, exists := w.items[itemID]; !exists {
		return fmt.Errorf("item %q does not exist", itemID)
	}

	inventory := w.playerInventories[playerName]

	// Find one instance of the requested definition in the inventory.
	instanceIndex := -1
	var selectedInstanceID string

	for i, instanceID := range inventory {
		instance, exists := w.itemInstances[instanceID]
		if !exists {
			return fmt.Errorf(
				"inventory references nonexistent instance %q",
				instanceID,
			)
		}

		if instance.ItemID == itemID {
			instanceIndex = i
			selectedInstanceID = instanceID
			break
		}
	}

	if instanceIndex == -1 {
		return fmt.Errorf(
			"player %q does not have item %q",
			playerName,
			itemID,
		)
	}

	item := w.items[itemID]

	// Only common items can be dropped.
	if item.Type != ItemTypeCommon {
		return fmt.Errorf(
			"item %q cannot be dropped because its type is %q",
			itemID,
			item.Type,
		)
	}

	// Remove the instance from the player's inventory.
	inventory = append(
		inventory[:instanceIndex],
		inventory[instanceIndex+1:]...,
	)

	if len(inventory) == 0 {
		delete(w.playerInventories, playerName)
	} else {
		w.playerInventories[playerName] = inventory
	}

	// Place that same instance in the room.
	w.roomItemInstances[roomID] = append(
		w.roomItemInstances[roomID],
		selectedInstanceID,
	)

	return nil
}

// GetInventory returns the item definitions in a player's inventory.
// Each entry corresponds to one concrete instance.
func (w *World) GetInventory(playerName string) ([]Item, error) {
	if w == nil {
		return nil, fmt.Errorf("world is nil")
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if _, exists := w.playerRooms[playerName]; !exists {
		return nil, fmt.Errorf("player %q is not in the world", playerName)
	}

	inventory := w.playerInventories[playerName]
	items := make([]Item, 0, len(inventory))

	for _, instanceID := range inventory {
		instance, exists := w.itemInstances[instanceID]
		if !exists {
			return nil, fmt.Errorf(
				"inventory references nonexistent instance %q",
				instanceID,
			)
		}

		item, exists := w.items[instance.ItemID]
		if !exists {
			return nil, fmt.Errorf(
				"instance %q references nonexistent item %q",
				instanceID,
				instance.ItemID,
			)
		}

		items = append(items, item)
	}

	return items, nil
}