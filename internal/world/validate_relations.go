package world

import "fmt"

// validateWorldRelations checks cross-references between world entities.
func (w *World) validateWorldRelations() error {

	if err := w.validateQuestNPCRelations(); err != nil {
		return err
	}

	if err := w.validateNPCRoomRelations(); err != nil {
		return err
	}

	if err := w.validateNPCQuestDialogueRelations(); err != nil {
		return err
	}

	return nil
}

// validateQuestNPCRelations checks the bidirectional relationship
// between quests and their giver NPCs.
func (w *World) validateQuestNPCRelations() error {
	for questID, quest := range w.quests {
		npc, exists := w.npcs[quest.GiverNPCID]
		if !exists {
			// The individual quest validator already checks this reference.
			continue
		}

		if npc.QuestID != questID {
			return fmt.Errorf(
				"quest %q references giver NPC %q, but NPC references quest %q",
				questID,
				quest.GiverNPCID,
				npc.QuestID,
			)
		}
	}

	for npcID, npc := range w.npcs {
		if npc.QuestID == "" {
			continue
		}

		quest, exists := w.quests[npc.QuestID]
		if !exists {
			return fmt.Errorf(
				"NPC %q references nonexistent quest %q",
				npcID,
				npc.QuestID,
			)
		}

		if quest.GiverNPCID != npcID {
			return fmt.Errorf(
				"NPC %q references quest %q, but that quest belongs to NPC %q",
				npcID,
				npc.QuestID,
				quest.GiverNPCID,
			)
		}
	}

	return nil
}

// validateNPCRoomRelations checks that every NPC is assigned to a room.
func (w *World) validateNPCRoomRelations() error {
	assignedNPCs := make(map[string]struct{})

	for _, room := range w.rooms {
		for _, npcID := range room.NPCs {
			assignedNPCs[npcID] = struct{}{}
		}
	}

	for npcID := range w.npcs {
		if _, exists := assignedNPCs[npcID]; !exists {
			return fmt.Errorf(
				"NPC %q is not assigned to any room",
				npcID,
			)
		}
	}

	return nil
}

// validateNPCQuestDialogueRelations checks NPC quest dialogue references.
func (w *World) validateNPCQuestDialogueRelations() error {
	for npcID, npc := range w.npcs {
		for questID := range npc.QuestDialogues {
			if questID == "" {
				return fmt.Errorf(
					"NPC %q has an empty quest dialogue ID",
					npcID,
				)
			}

			if _, exists := w.quests[questID]; !exists {
				return fmt.Errorf(
					"NPC %q references nonexistent quest %q in quest dialogues",
					npcID,
					questID,
				)
			}
		}
	}

	return nil
}