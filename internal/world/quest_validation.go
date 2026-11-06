package world

import (
	"fmt"
	"strings"
)

// validateQuests checks that quest definitions and their references are valid.
func (w *World) validateQuests() error {
	for key, quest := range w.quests {
		if err := validateQuestIdentity(key, quest); err != nil {
			return err
		}

		if err := w.validateQuestGiver(key, quest.GiverNPCID); err != nil {
			return err
		}

		if err := w.validateQuestRequiredItem(key, quest); err != nil {
			return err
		}

		if err := w.validateQuestRewards(key, quest.RewardItemIDs); err != nil {
			return err
		}

		if err := w.validateQuestNextQuest(key, quest); err != nil {
			return err
		}
	}

	return w.validateQuestChains()
}

// validateQuestIdentity checks the quest ID and name.
func validateQuestIdentity(key string, quest Quest) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("quest map contains an empty key")
	}

	if strings.TrimSpace(quest.ID) == "" {
		return fmt.Errorf("quest %q has an empty ID", key)
	}

	if quest.ID != key {
		return fmt.Errorf(
			"quest key %q does not match quest ID %q",
			key,
			quest.ID,
		)
	}

	if strings.TrimSpace(quest.Name) == "" {
		return fmt.Errorf("quest %q has an empty name", key)
	}

	return nil
}

// validateQuestGiver checks that the quest giver exists.
func (w *World) validateQuestGiver(key, giverNPCID string) error {
	if strings.TrimSpace(giverNPCID) == "" {
		return fmt.Errorf("quest %q has an empty giver NPC ID", key)
	}

	if _, exists := w.npcs[giverNPCID]; !exists {
		return fmt.Errorf(
			"quest %q references nonexistent giver NPC %q",
			key,
			giverNPCID,
		)
	}

	return nil
}

// validateQuestRequiredItem checks the required item reference and quantity.
func (w *World) validateQuestRequiredItem(key string, quest Quest) error {
	if strings.TrimSpace(quest.RequiredItemID) == "" {
		if quest.RequiredItemQuantity != 0 {
			return fmt.Errorf(
				"quest %q has required item quantity %d but no required item ID",
				key,
				quest.RequiredItemQuantity,
			)
		}

		return nil
	}

	if quest.RequiredItemQuantity <= 0 {
		return fmt.Errorf(
			"quest %q has invalid required item quantity %d",
			key,
			quest.RequiredItemQuantity,
		)
	}

	if _, exists := w.items[quest.RequiredItemID]; !exists {
		return fmt.Errorf(
			"quest %q references nonexistent required item %q",
			key,
			quest.RequiredItemID,
		)
	}

	return nil
}

// validateQuestRewards checks that all quest reward references exist.
func (w *World) validateQuestRewards(key string, rewardItemIDs []string) error {
	for _, itemID := range rewardItemIDs {
		if strings.TrimSpace(itemID) == "" {
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
	}

	return nil
}

// validateQuestNextQuest checks that the next quest reference is valid.
func (w *World) validateQuestNextQuest(key string, quest Quest) error {
	if strings.TrimSpace(quest.NextQuestID) == "" {
		return nil
	}

	if _, exists := w.quests[quest.NextQuestID]; !exists {
		return fmt.Errorf(
			"quest %q references nonexistent next quest %q",
			key,
			quest.NextQuestID,
		)
	}

	return nil
}

// validateQuestChains checks that quest chains do not contain cycles.
func (w *World) validateQuestChains() error {
	visited := make(map[string]struct{})
	inPath := make(map[string]struct{})

	var visit func(string) error

	visit = func(questID string) error {
		if _, exists := inPath[questID]; exists {
			return fmt.Errorf(
				"quest chain contains a cycle involving quest %q",
				questID,
			)
		}

		if _, exists := visited[questID]; exists {
			return nil
		}

		visited[questID] = struct{}{}
		inPath[questID] = struct{}{}

		quest := w.quests[questID]

		if quest.NextQuestID != "" {
			if err := visit(quest.NextQuestID); err != nil {
				return err
			}
		}

		delete(inPath, questID)

		return nil
	}

	for questID := range w.quests {
		if err := visit(questID); err != nil {
			return err
		}
	}

	return nil
}