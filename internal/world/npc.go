package world

type NPCSummary struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Role        string      `json:"role"`
	Hostile     bool        `json:"hostile"`
	CombatStats CombatStats `json:"combat_stats"`
}

// NPCDialogue contains the dialogue lines for each quest state.
type NPCDialogue struct {
	Default   []string `json:"default"`
	Available []string `json:"available"`
	Active    []string `json:"active"`
	Completed []string `json:"completed"`
	Locked    []string `json:"locked"`
}

// NPC represents a non-player character in the game world.
type NPC struct {
	ID             string                  `json:"id"`
	Name           string                  `json:"name"`
	Role           string                  `json:"role"`
	Hostile        bool                    `json:"hostile"`
	CombatStats    CombatStats             `json:"combat_stats"`
	QuestID        string                  `json:"quest_id"`
	Dialogue       NPCDialogue             `json:"dialogue"`
	QuestDialogues map[string]NPCDialogue  `json:"quest_dialogues"`
	Rewards 		[]string `json:"rewards,omitempty"`
}

// NPCConversation represents a conversation between a player and an NPC.
type NPCConversation struct {
	NPC         NPCSummary  `json:"npc"`
	Dialogue    []string    `json:"dialogue"`
	Quest       *QuestSummary `json:"quest,omitempty"`
	QuestStatus QuestStatus `json:"quest_status,omitempty"`
}


// dialogueForQuestStatus selects dialogue lines for a specific quest state.
func dialogueForQuestStatus(
	dialogue NPCDialogue,
	status QuestStatus,
) []string {
	var lines []string

	switch status {
	case QuestStatusAvailable:
		lines = dialogue.Available
	case QuestStatusActive:
		lines = dialogue.Active
	case QuestStatusCompleted:
		lines = dialogue.Completed
	case QuestStatusLocked:
		lines = dialogue.Locked
	}

	// If this state has no dialogue, fall back to the NPC's default lines.
	if len(lines) == 0 {
		lines = dialogue.Default
	}

	return cloneSlice(lines)
}