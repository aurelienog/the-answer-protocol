package world

import "fmt"

// validateQuests checks that quest definitions and their references are valid.
func (w *World) validateQuests() error {
	for key, quest := range w.quests {
		if key == "" || quest.ID == "" {
			return fmt.Errorf("quest ID cannot be empty")
		}

		if quest.ID != key {
			return fmt.Errorf(
				"quest key %q does not match quest ID %q",
				key,
				quest.ID,
			)
		}

		if quest.Name == "" {
			return fmt.Errorf("quest %q has an empty name", key)
		}

		if quest.GiverNPCID == "" {
			return fmt.Errorf("quest %q has an empty giver NPC ID", key)
		}

		if _, exists := w.npcs[quest.GiverNPCID]; !exists {
			return fmt.Errorf(
				"quest %q references nonexistent giver NPC %q",
				key,
				quest.GiverNPCID,
			)
		}

		if quest.RequiredItemID != "" {
			if _, exists := w.items[quest.RequiredItemID]; !exists {
				return fmt.Errorf(
					"quest %q references nonexistent required item %q",
					key,
					quest.RequiredItemID,
				)
			}
		}

		seenRewards := make(map[string]struct{})

		for _, itemID := range quest.RewardItemIDs {
			if itemID == "" {
				return fmt.Errorf(
					"quest %q has an empty reward item ID",
					key,
				)
			}

			if _, exists := w.items[itemID]; !exists {
				return fmt.Errorf(
					"quest %q references nonexistent reward item %q",
					key,
					itemID,
				)
			}

			if _, exists := seenRewards[itemID]; exists {
				return fmt.Errorf(
					"quest %q lists reward item %q more than once",
					key,
					itemID,
				)
			}

			seenRewards[itemID] = struct{}{}
		}

		if quest.NextQuestID != "" {
			if _, exists := w.quests[quest.NextQuestID]; !exists {
				return fmt.Errorf(
					"quest %q references nonexistent next quest %q",
					key,
					quest.NextQuestID,
				)
			}

			if quest.NextQuestID == quest.ID {
				return fmt.Errorf(
					"quest %q cannot reference itself as the next quest",
					key,
				)
			}
		}
	}

	return nil
}