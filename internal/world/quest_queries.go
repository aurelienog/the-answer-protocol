package world

import (
	"fmt"
	"sort"
)

// GetPlayerQuest returns a player's progress for a quest.
func (w *World) GetPlayerQuest(
	playerName string,
	questID string,
) (PlayerQuest, error) {
	if w == nil {
		return PlayerQuest{}, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	if playerName == "" || questID == "" {
		return PlayerQuest{}, fmt.Errorf(
			"%w: player name and quest ID cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if _, exists := w.quests[questID]; !exists {
		return PlayerQuest{}, fmt.Errorf(
			"%w: %q",
			ErrQuestNotFound,
			questID,
		)
	}

	playerQuests, exists := w.playerQuests[playerName]
	if !exists {
		return PlayerQuest{}, fmt.Errorf(
			"%w: player %q has no quest progress",
			ErrNoQuestAvailable,
			playerName,
		)
	}

	playerQuest, exists := playerQuests[questID]
	if !exists {
		return PlayerQuest{}, fmt.Errorf(
			"%w: player %q has no progress for quest %q",
			ErrNoQuestAvailable,
			playerName,
			questID,
		)
	}

	return playerQuest, nil
}

// GetAvailableQuests returns quests that a player can currently accept.
func (w *World) GetAvailableQuests(playerName string) ([]Quest, error) {
	if w == nil {
		return nil, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	if playerName == "" {
		return nil, fmt.Errorf("%w: player name cannot be empty", ErrInvalidArgument)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if _, exists := w.playerRooms[playerName]; !exists {
		return nil, fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
	}

	playerQuests := w.playerQuests[playerName]
	available := make([]Quest, 0)

	for questID, quest := range w.quests {
		// Skip quests that have already been accepted or completed.
		if progress, exists := playerQuests[questID]; exists {
			if progress.Status == QuestStatusActive ||
				progress.Status == QuestStatusCompleted {
				continue
			}
		}

		// A quest with a predecessor is available only after that
		// predecessor has been completed.
		predecessorCompleted := true

		for previousQuestID, previousQuest := range w.quests {
			if previousQuest.NextQuestID != questID {
				continue
			}

			progress, exists := playerQuests[previousQuestID]
			if !exists || progress.Status != QuestStatusCompleted {
				predecessorCompleted = false
				break
			}
		}

		if predecessorCompleted {
			quest.RewardItemIDs = cloneSlice(quest.RewardItemIDs)
			available = append(available, quest)
		}
	}

	// Map iteration order is unspecified; sort by ID for deterministic results.
	sort.Slice(available, func(i, j int) bool {
		return available[i].ID < available[j].ID
	})

	return available, nil
}

// GetPlayerQuests returns the active and completed quests of a player.
func (w *World) GetPlayerQuests(
	playerName string,
) ([]PlayerQuestSummary, error) {
	if w == nil {
		return nil, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	if playerName == "" {
		return nil, fmt.Errorf(
			"%w: player name cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if _, exists := w.playerRooms[playerName]; !exists {
		return nil, fmt.Errorf(
			"%w: %q",
			ErrPlayerNotFound,
			playerName,
		)
	}

	playerQuests := w.playerQuests[playerName]
	quests := make([]PlayerQuestSummary, 0)

	for questID, progress := range playerQuests {
		if progress.Status != QuestStatusActive &&
			progress.Status != QuestStatusCompleted {
			continue
		}

		quest, exists := w.quests[questID]
		if !exists {
			return nil, fmt.Errorf(
				"%w: player %q references nonexistent quest %q",
				ErrInvalidWorldState,
				playerName,
				questID,
			)
		}

		quests = append(quests, PlayerQuestSummary{
			ID:          quest.ID,
			Name:        quest.Name,
			Description: quest.Description,
			Status:      progress.Status,
		})
	}

	sort.Slice(quests, func(i, j int) bool {
		return quests[i].ID < quests[j].ID
	})

	return quests, nil
}

// GetPlayerQuestSummary returns a quest's public information only if
// the player has accepted or completed it.
func (w *World) GetPlayerQuestSummary(
	playerName, questID string,
) (PlayerQuestSummary, error) {
	if w == nil {
		return PlayerQuestSummary{}, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	if playerName == "" || questID == "" {
		return PlayerQuestSummary{}, fmt.Errorf(
			"%w: player name and quest ID cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if _, exists := w.playerRooms[playerName]; !exists {
		return PlayerQuestSummary{}, fmt.Errorf(
			"%w: %q",
			ErrPlayerNotFound,
			playerName,
		)
	}

	quest, exists := w.quests[questID]
	if !exists {
		return PlayerQuestSummary{}, fmt.Errorf(
			"%w: %q",
			ErrQuestNotFound,
			questID,
		)
	}

	progress, exists := w.playerQuests[playerName][questID]
	if !exists ||
		(progress.Status != QuestStatusActive &&
			progress.Status != QuestStatusCompleted) {
		return PlayerQuestSummary{}, fmt.Errorf(
			"%w: player %q cannot access quest %q",
			ErrNoQuestAvailable,
			playerName,
			questID,
		)
	}

	return PlayerQuestSummary{
		ID:          quest.ID,
		Name:        quest.Name,
		Description: quest.Description,
		Status:      progress.Status,
	}, nil
}
