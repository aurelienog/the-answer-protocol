package world

// NPC represents a non-player character in the game world.
type NPC struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Role     string   `json:"role"`
	Hostile  bool     `json:"hostile"`
	HP       int      `json:"hp"`
	MaxHP    int      `json:"max_hp"`
	Damage   int      `json:"damage"`
	QuestID  string   `json:"quest_id"`
	Dialogue []string `json:"dialogue"`
}

// NPCConversation represents a conversation between a player and an NPC.
type NPCConversation struct {
	NPC      NPC           `json:"npc"`
	Dialogue []string      `json:"dialogue"`
	Quest    *Quest        `json:"quest,omitempty"`
	QuestStatus QuestStatus `json:"quest_status,omitempty"`
}