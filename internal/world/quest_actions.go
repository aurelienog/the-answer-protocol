package world

import "fmt"

// AcceptQuest accepts a quest for a registered player.
func (w *World) AcceptQuest(playerName, questID string) error {
	if w == nil {
		return fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	if playerName == "" || questID == "" {
		return fmt.Errorf(
			"%w: player name and quest ID cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	player, exists := w.players[playerName]
	if !exists {
		return fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
	}

	quest, exists := w.quests[questID]
	if !exists {
		return fmt.Errorf("%w: %q", ErrQuestNotFound, questID)
	}
	
	// The player must have discovered the quest by talking to its giver.
	playerQuests := w.playerQuests[playerName]
	progress, discovered := playerQuests[questID]

	if !discovered || progress.Status != QuestStatusAvailable {
		return fmt.Errorf(
			"%w: quest %q has not been discovered or is not available",
			ErrNoQuestAvailable,
			questID,
		)
	}

	// The quest giver must be present in the player's current room.
	roomID := player.RoomID
	room, exists := w.rooms[roomID]
	if !exists {
		return fmt.Errorf(
			"%w: player %q references nonexistent room %q",
			ErrInvalidWorldState,
			playerName,
			roomID,
		)
	}

	giverPresent := false
	for _, npcID := range room.NPCs {
		if npcID == quest.GiverNPCID {
			giverPresent = true
			break
		}
	}

	if !giverPresent {
		return fmt.Errorf(
			"%w: quest giver %q is not in the player's room",
			ErrNoQuestAvailable,
			quest.GiverNPCID,
		)
	}

	// Validate the quest's prerequisites.
	canAccept, reason := w.canAcceptQuestLocked(playerName, questID)
	if !canAccept {
		return fmt.Errorf(
			"%w: quest %q: %s",
			ErrNoQuestAvailable,
			questID,
			reason,
		)
	}

	if w.playerQuests == nil {
		w.playerQuests = make(map[string]map[string]PlayerQuest)
	}

	if w.playerQuests[playerName] == nil {
		w.playerQuests[playerName] = make(map[string]PlayerQuest)
	}

	w.playerQuests[playerName][questID] = PlayerQuest{
		QuestID: quest.ID,
		Status:  QuestStatusActive,
	}

	return nil
}

// canAcceptQuestLocked checks whether a player can accept a quest.
// The caller must hold w.mu.Lock() or w.mu.RLock().
func (w *World) canAcceptQuestLocked(playerName, questID string) (bool, string) {
	if playerQuests, exists := w.playerQuests[playerName]; exists {
		if progress, exists := playerQuests[questID]; exists {
			switch progress.Status {
			case QuestStatusActive:
				return false, "quest is already active"
			case QuestStatusCompleted:
				return false, "quest is already completed"
			}
		}
	}

	// A quest with a predecessor requires that predecessor to be completed.
	for previousQuestID, previousQuest := range w.quests {
		if previousQuest.NextQuestID != questID {
			continue
		}

		playerQuests := w.playerQuests[playerName]
		progress, exists := playerQuests[previousQuestID]

		if !exists || progress.Status != QuestStatusCompleted {
			return false, fmt.Sprintf(
				"quest requires completion of quest %q",
				previousQuestID,
			)
		}
	}

	return true, ""
}

// CompleteQuest completes an active quest for a player.
// It consumes the required item when configured and grants the rewards.
func (w *World) CompleteQuest(playerName, questID string) error {
	if w == nil {
		return fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	if playerName == "" || questID == "" {
		return fmt.Errorf(
			"%w: player name and quest ID cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	player, exists := w.players[playerName]
	if !exists {
		return fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
	}

	roomID := player.RoomID

	room, exists := w.rooms[roomID]
	if !exists {
		return fmt.Errorf(
			"%w: player %q references nonexistent room %q",
			ErrInvalidWorldState,
			playerName,
			roomID,
		)
	}

	quest, exists := w.quests[questID]
	if !exists {
		return fmt.Errorf("%w: %q", ErrQuestNotFound, questID)
	}

	playerQuests := w.playerQuests[playerName]
	progress, exists := playerQuests[questID]
	if !exists || progress.Status != QuestStatusActive {
		return fmt.Errorf(
			"%w: quest %q is not active for player %q",
			ErrNoQuestAvailable,
			questID,
			playerName,
		)
	}

	// The quest giver must be in the player's current room.
	giverPresent := false
	for _, npcID := range room.NPCs {
		if npcID == quest.GiverNPCID {
			giverPresent = true
			break
		}
	}

	if !giverPresent {
		return fmt.Errorf(
			"%w: quest giver %q is not in the player's room",
			ErrNoQuestAvailable,
			quest.GiverNPCID,
		)
	}

	// Find the required item in the player's inventory, if any.
	inventory := w.playerInventories[playerName]
	requiredItemIndex := -1

	if quest.RequiredItemID != "" {
		for i, instanceID := range inventory {
			instance, exists := w.itemInstances[instanceID]
			if !exists {
				return fmt.Errorf(
					"%w: inventory references nonexistent instance %q",
					ErrInvalidWorldState,
					instanceID,
				)
			}

			if _, exists := w.items[instance.ItemID]; !exists {
				return fmt.Errorf(
					"%w: instance %q references nonexistent item %q",
					ErrInvalidWorldState,
					instanceID,
					instance.ItemID,
				)
			}

			if instance.ItemID == quest.RequiredItemID {
				requiredItemIndex = i
				break
			}
		}

		if requiredItemIndex == -1 {
			return fmt.Errorf(
				"%w: player %q does not have required item %q",
				ErrItemNotInInventory,
				playerName,
				quest.RequiredItemID,
			)
		}
	}

	// Validate all reward definitions before changing the world.
	for _, rewardItemID := range quest.RewardItemIDs {
		if _, exists := w.items[rewardItemID]; !exists {
			return fmt.Errorf(
				"%w: quest %q references nonexistent reward item %q",
				ErrInvalidWorldState,
				questID,
				rewardItemID,
			)
		}
	}

	// Prepare unique reward instance IDs without registering instances yet.
	rewardInstances := make([]ItemInstance, 0, len(quest.RewardItemIDs))
	nextID := w.nextItemInstanceID

	for _, rewardItemID := range quest.RewardItemIDs {
		var instanceID string

		for {
			nextID++
			instanceID = fmt.Sprintf("instance:reward:%d", nextID)

			if _, exists := w.itemInstances[instanceID]; !exists {
				break
			}
		}

		rewardInstances = append(rewardInstances, ItemInstance{
			ID:     instanceID,
			ItemID: rewardItemID,
		})
	}

	// Apply the changes only after validation succeeds.
	if quest.ConsumeRequiredItem && requiredItemIndex >= 0 {
		inventory = append(
			inventory[:requiredItemIndex],
			inventory[requiredItemIndex+1:]...,
		)
	}

	for _, instance := range rewardInstances {
		w.itemInstances[instance.ID] = instance
		inventory = append(inventory, instance.ID)
	}

	if len(inventory) == 0 {
		delete(w.playerInventories, playerName)
	} else {
		w.playerInventories[playerName] = inventory
	}

	w.nextItemInstanceID = nextID
	progress.Status = QuestStatusCompleted
	w.playerQuests[playerName][questID] = progress

	return nil
}