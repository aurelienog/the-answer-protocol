package world

// Quest defines a mission available in the game world.
type Quest struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	GiverNPCID          string   `json:"giver_npc_id"`
	RequiredItemID      string   `json:"required_item_id"`
	ConsumeRequiredItem bool     `json:"consume_required_item"`
	RewardItemIDs       []string `json:"reward_item_ids"`
	NextQuestID         string   `json:"next_quest_id"`
}

// QuestStatus represents a player's progress on a quest.
type QuestStatus string

const (
	QuestStatusAvailable QuestStatus = "available"
	QuestStatusActive    QuestStatus = "active"
	QuestStatusCompleted QuestStatus = "completed"
	QuestStatusLocked    QuestStatus = "locked"
)

// QuestSummary contains the public information about a quest
// that is returned to the client during an NPC conversation.
type QuestSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// PlayerQuest represents the progress of one player on one quest.
type PlayerQuest struct {
	QuestID string
	Status  QuestStatus
}