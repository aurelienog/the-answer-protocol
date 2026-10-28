package world

import (
	"sync"
)

// World represents the game world.
type World struct {
	mu                sync.RWMutex
	rooms             map[string]Room
	playerRooms       map[string]string
	items             map[string]Item
	itemInstances     map[string]ItemInstance
	roomItemInstances map[string][]string
	playerInventories map[string][]string
}

// Room represents a location in the game world.
type Room struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Exits       map[string]string `json:"exits"`
	Items       map[string]int `json:"items"`
	NPCs        []string       `json:"npcs"`
}

// cloneMap returns a copy of a string map.
func cloneMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}

	result := make(map[string]string, len(source))

	for key, value := range source {
		result[key] = value
	}

	return result
}

// cloneSlice returns a copy of a string slice.
func cloneSlice(source []string) []string {
	if source == nil {
		return nil
	}

	result := make([]string, len(source))
	copy(result, source)

	return result
}

// cloneItemCounts returns a copy of an item-count map.
func cloneItemCounts(source map[string]int) map[string]int {
	if source == nil {
		return nil
	}

	result := make(map[string]int, len(source))

	for key, value := range source {
		result[key] = value
	}

	return result
}