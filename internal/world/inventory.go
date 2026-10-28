package world

import "fmt"

// TakeItem moves one concrete item instance from the player's current room
// to their inventory.
func (w *World) TakeItem(playerName, itemID string) error {
	if w == nil {
		return fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	roomID, exists := w.playerRooms[playerName]
	if !exists {
		return fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
	}

	if _, exists := w.rooms[roomID]; !exists {
		return fmt.Errorf("%w: room %q does not exist", ErrInvalidWorldState, roomID)
	}

	if _, exists := w.items[itemID]; !exists {
		return fmt.Errorf("%w: %q", ErrItemNotFound, itemID)
	}

	// Prevent the player from acquiring the same item definition twice.
	for _, instanceID := range w.playerInventories[playerName] {
		instance, exists := w.itemInstances[instanceID]
		if !exists {
			return fmt.Errorf(
				"%w: inventory references nonexistent instance %q",
				ErrInvalidWorldState,
				instanceID,
			)
		}

		if _, exists := w.items[instance.ItemID]; !exists {
			return fmt.Errorf(
				"%w: instance %q references nonexistent item %q",
				ErrInvalidWorldState,
				instanceID,
				instance.ItemID,
			)
		}

		if instance.ItemID == itemID {
			return fmt.Errorf(
				"%w: player %q already has item %q",
				ErrItemAlreadyOwned,
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
				"%w: room %q references nonexistent instance %q",
				ErrInvalidWorldState,
				roomID,
				instanceID,
			)
		}

		if _, exists := w.items[instance.ItemID]; !exists {
			return fmt.Errorf(
				"%w: instance %q references nonexistent item %q",
				ErrInvalidWorldState,
				instanceID,
				instance.ItemID,
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
			"%w: item %q is not in room %q",
			ErrItemNotInRoom,
			itemID,
			roomID,
		)
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
		return fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	roomID, exists := w.playerRooms[playerName]
	if !exists {
		return fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
	}

	if _, exists := w.rooms[roomID]; !exists {
		return fmt.Errorf("%w: room %q does not exist", ErrInvalidWorldState, roomID)
	}

	item, exists := w.items[itemID]
	if !exists {
		return fmt.Errorf("%w: %q", ErrItemNotFound, itemID)
	}

	inventory := w.playerInventories[playerName]

	// Find one instance of the requested definition in the inventory.
	instanceIndex := -1
	var selectedInstanceID string

	for i, instanceID := range inventory {
		instance, exists := w.itemInstances[instanceID]
		if !exists {
			return fmt.Errorf(
				"%w: inventory references nonexistent instance %q",
				ErrInvalidWorldState,
				instanceID,
			)
		}

		if _, exists := w.items[instance.ItemID]; !exists {
			return fmt.Errorf(
				"%w: instance %q references nonexistent item %q",
				ErrInvalidWorldState,
				instanceID,
				instance.ItemID,
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
			"%w: player %q does not have item %q",
			ErrItemNotInInventory,
			playerName,
			itemID,
		)
	}

	// Only common items can be dropped.
	if item.Type != ItemTypeCommon {
		return fmt.Errorf(
			"%w: item %q has type %q",
			ErrItemCannotBeDropped,
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
		return nil, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if _, exists := w.playerRooms[playerName]; !exists {
		return nil, fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
	}

	inventory := w.playerInventories[playerName]
	items := make([]Item, 0, len(inventory))

	for _, instanceID := range inventory {
		instance, exists := w.itemInstances[instanceID]
		if !exists {
			return nil, fmt.Errorf(
				"%w: inventory references nonexistent instance %q",
				ErrInvalidWorldState,
				instanceID,
			)
		}

		item, exists := w.items[instance.ItemID]
		if !exists {
			return nil, fmt.Errorf(
				"%w: instance %q references nonexistent item %q",
				ErrInvalidWorldState,
				instanceID,
				instance.ItemID,
			)
		}

		items = append(items, item)
	}

	return items, nil
}