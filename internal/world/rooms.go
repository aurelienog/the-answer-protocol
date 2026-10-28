package world

import "fmt"

// GetRoom returns a snapshot of a room.
func (w *World) GetRoom(roomID string) (Room, error) {
	if w == nil {
		return Room{}, fmt.Errorf("world is nil")
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	room, exists := w.rooms[roomID]
	if !exists {
		return Room{}, fmt.Errorf("room %q does not exist", roomID)
	}

	// Return copies of the collections so callers cannot
	// modify the world's internal state through the result.
	room.Exits = cloneMap(room.Exits)
	room.Items = cloneSlice(room.Items)
	room.NPCs = cloneSlice(room.NPCs)

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

// GetRoomItems returns the complete item definitions for a room.
func (w *World) GetRoomItems(roomID string) ([]Item, error) {
	if w == nil {
		return nil, fmt.Errorf("world is nil")
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	room, exists := w.rooms[roomID]
	if !exists {
		return nil, fmt.Errorf("room %q does not exist", roomID)
	}

	items := make([]Item, 0, len(room.Items))

	for _, itemID := range room.Items {
		item, exists := w.items[itemID]
		if !exists {
			return nil, fmt.Errorf(
				"room %q references nonexistent item %q",
				roomID,
				itemID,
			)
		}

		items = append(items, item)
	}

	return items, nil
}