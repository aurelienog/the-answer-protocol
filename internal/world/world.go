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
	playerInventories map[string][]string
}

// Room represents a location in the game world.
type Room struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Exits       map[string]string `json:"exits"`
	Items       []string          `json:"items"`
	NPCs        []string          `json:"npcs"`
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

// cloneMap returns a copy of a string map.
func cloneSlice(source []string) []string {
	if source == nil {
		return nil
	}

	result := make([]string, len(source))
	copy(result, source)

	return result
}
