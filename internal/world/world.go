package world

// World represents the game world.
type World struct {
	Rooms map[string]Room `json:"rooms"`
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

