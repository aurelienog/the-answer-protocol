package world

import (
	"fmt"
	"strings"
)

// GetRoom returns a snapshot of a room with current item quantities.
func (w *World) GetRoom(roomID string) (Room, error) {
	if w == nil {
		return Room{}, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	room, exists := w.rooms[roomID]
	if !exists {
		return Room{}, fmt.Errorf(
			"%w: room %q does not exist",
			ErrInvalidWorldState,
			roomID,
		)
	}

	room.Exits = cloneMap(room.Exits)
	room.NPCs = cloneSlice(room.NPCs)

	// Derive item quantities from the current room instances.
	room.Items = make(map[string]int)

	for _, instanceID := range w.roomItemInstances[roomID] {
		instance, exists := w.itemInstances[instanceID]
		if !exists {
			return Room{}, fmt.Errorf(
				"%w: room %q references nonexistent instance %q",
				ErrInvalidWorldState,
				roomID,
				instanceID,
			)
		}

		if _, exists := w.items[instance.ItemID]; !exists {
			return Room{}, fmt.Errorf(
				"%w: instance %q references nonexistent item %q",
				ErrInvalidWorldState,
				instanceID,
				instance.ItemID,
			)
		}

		room.Items[instance.ItemID]++
	}

	return room, nil
}

// RoomCount returns the number of rooms in the world.
func (w *World) RoomCount() int {
	if w == nil {
		return 0
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	return len(w.rooms)
}


// GetRoomItems returns the item definitions for all instances in a room.
func (w *World) GetRoomItems(roomID string) ([]Item, error) {
	if w == nil {
		return nil, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if _, exists := w.rooms[roomID]; !exists {
		return nil, fmt.Errorf(
			"%w: room %q does not exist",
			ErrInvalidWorldState,
			roomID,
		)
	}

	instanceIDs := w.roomItemInstances[roomID]
	items := make([]Item, 0, len(instanceIDs))

	for _, instanceID := range instanceIDs {
		instance, exists := w.itemInstances[instanceID]
		if !exists {
			return nil, fmt.Errorf(
				"%w: room %q references nonexistent instance %q",
				ErrInvalidWorldState,
				roomID,
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

// FindRoomItem finds an available item definition by canonical ID or display name.
// Matching is case-insensitive and ignores surrounding whitespace.
func (w *World) FindRoomItem(roomID, query string) (Item, error) {
	if w == nil {
		return Item{}, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return Item{}, fmt.Errorf("%w: item name cannot be empty", ErrInvalidArgument)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if _, exists := w.rooms[roomID]; !exists {
		return Item{}, fmt.Errorf(
			"%w: room %q does not exist",
			ErrInvalidWorldState,
			roomID,
		)
	}

	normalizedQuery := strings.ToLower(query)

	var match Item
	found := false

	for _, instanceID := range w.roomItemInstances[roomID] {
		instance, exists := w.itemInstances[instanceID]
		if !exists {
			return Item{}, fmt.Errorf(
				"%w: room %q references nonexistent instance %q",
				ErrInvalidWorldState,
				roomID,
				instanceID,
			)
		}

		item, exists := w.items[instance.ItemID]
		if !exists {
			return Item{}, fmt.Errorf(
				"%w: instance %q references nonexistent item %q",
				ErrInvalidWorldState,
				instanceID,
				instance.ItemID,
			)
		}

		if !strings.EqualFold(item.ID, normalizedQuery) &&
			!strings.EqualFold(item.Name, normalizedQuery) {
			continue
		}

		if found && match.ID != item.ID {
			return Item{}, fmt.Errorf(
				"%w: item name %q is ambiguous in room %q",
				ErrItemNameAmbiguous,
				query,
				roomID,
			)
		}

		match = item
		found = true
	}

	if !found {
		return Item{}, fmt.Errorf(
			"%w: item %q is not available in room %q",
			ErrItemNotFound,
			query,
			roomID,
		)
	}

	return match, nil
}

