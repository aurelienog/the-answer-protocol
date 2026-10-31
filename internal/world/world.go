package world

import (
	"sync"
)

const InitialRoomID = "harbor_square"

// World represents the game world.
type World struct {
	mu sync.RWMutex

	rooms   map[string]Room
	players map[string]*Player

	items             map[string]Item
	itemInstances     map[string]ItemInstance
	roomItemInstances map[string][]string

	playerInventories map[string][]string
	playerQuests      map[string]map[string]PlayerQuest

	npcs   map[string]NPC
	quests map[string]Quest

	groups       map[string]Group
	playerGroups map[string]string

	nextItemInstanceID uint64
	nextGroupID        uint64
	nextCombatID		uint64

	combats map[string]*Combat
}

// Room represents a location in the game world.
type Room struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Exits       map[string]string `json:"exits"`
	Items       map[string]int    `json:"items"`
	NPCs        []string          `json:"npcs"`
}

type Group struct {
	ID          string
	Name        string
	Members     []string
	Invitations map[string]bool
}

type GroupInvitation struct {
	ID   string
	Name string
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
