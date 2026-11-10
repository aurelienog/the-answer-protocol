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

	if w.playerInActiveCombatLocked(playerName) {
		return fmt.Errorf(
			"%w: player %q cannot accept quests during combat",
			ErrPlayerAlreadyInCombat,
			playerName,
		)
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
	quest, exists := w.quests[questID]
	if !exists {
		return false, "quest does not exist"
	}

	if playerQuests, exists := w.playerQuests[playerName]; exists {
		if progress, exists := playerQuests[questID]; exists {
			switch progress.Status {
			case QuestStatusActive:
				return false, "quest is already active"
			case QuestStatusCompleted:
				if !quest.Repeatable {
					return false, "quest is already completed"
				}
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

	if _, exists := w.players[playerName]; !exists {
		return fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
	}

	quest, exists := w.quests[questID]
	if !exists {
		return fmt.Errorf("%w: %q", ErrQuestNotFound, questID)
	}

	progress, exists := w.playerQuests[playerName][questID]
	if !exists || progress.Status != QuestStatusActive {
		return fmt.Errorf(
			"%w: quest %q is not active for player %q",
			ErrNoQuestAvailable,
			questID,
			playerName,
		)
	}

	room, exists := w.rooms[w.players[playerName].RoomID]
	if !exists {
		return fmt.Errorf(
			"%w: player %q references nonexistent room",
			ErrInvalidWorldState,
			playerName,
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

	resolved, err := w.resolveQuestLocked(playerName, quest)
	if err != nil {
		return err
	}

	if !resolved {
		return fmt.Errorf(
			"%w: player %q does not have %d instances of required item %q",
			ErrItemNotInInventory,
			playerName,
			quest.RequiredItemQuantity,
			quest.RequiredItemID,
		)
	}

	return nil
}

// resolveQuestLocked completes an active quest and grants its rewards.
// The caller must already hold w.mu.Lock().
func (w *World) resolveQuestLocked(
	playerName string,
	quest Quest,
) (bool, error) {
	playerQuests := w.playerQuests[playerName]
	progress, exists := playerQuests[quest.ID]

	if !exists || progress.Status != QuestStatusActive {
		return false, nil
	}

	if quest.RequiredItemID != "" && quest.RequiredItemQuantity <= 0 {
		return false, fmt.Errorf(
			"%w: quest %q has an invalid required item quantity",
			ErrInvalidWorldState,
			quest.ID,
		)
	}

	if quest.RequiredItemID == "" && quest.RequiredItemQuantity != 0 {
		return false, fmt.Errorf(
			"%w: quest %q has a quantity without a required item",
			ErrInvalidWorldState,
			quest.ID,
		)
	}

	inventory := w.playerInventories[playerName]
	requiredItemIndexes := make([]int, 0, quest.RequiredItemQuantity)

	if quest.RequiredItemID != "" {
		for i, instanceID := range inventory {
			instance, exists := w.itemInstances[instanceID]
			if !exists {
				return false, fmt.Errorf(
					"%w: inventory references nonexistent instance %q",
					ErrInvalidWorldState,
					instanceID,
				)
			}

			if _, exists := w.items[instance.ItemID]; !exists {
				return false, fmt.Errorf(
					"%w: instance %q references nonexistent item %q",
					ErrInvalidWorldState,
					instanceID,
					instance.ItemID,
				)
			}

			if instance.ItemID == quest.RequiredItemID {
				requiredItemIndexes = append(requiredItemIndexes, i)

				if len(requiredItemIndexes) == quest.RequiredItemQuantity {
					break
				}
			}
		}

		if len(requiredItemIndexes) < quest.RequiredItemQuantity {
			return false, nil
		}
	}

	// Validate all reward definitions before modifying the world.
	for _, rewardItemID := range quest.RewardItemIDs {
		if _, exists := w.items[rewardItemID]; !exists {
			return false, fmt.Errorf(
				"%w: quest %q references nonexistent reward item %q",
				ErrInvalidWorldState,
				quest.ID,
				rewardItemID,
			)
		}
	}

	// Prepare reward instances before applying changes.
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

	// Apply changes after validation succeeds.
	if quest.ConsumeRequiredItem && len(requiredItemIndexes) > 0 {
		requiredIndexes := make(map[int]bool, len(requiredItemIndexes))

		for _, index := range requiredItemIndexes {
			requiredIndexes[index] = true
		}

		remainingInventory := make([]string, 0, len(inventory)-len(requiredItemIndexes))

		for i, instanceID := range inventory {
			if !requiredIndexes[i] {
				remainingInventory = append(remainingInventory, instanceID)
			}
		}

		inventory = remainingInventory
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
	w.playerQuests[playerName][quest.ID] = progress

	return true, nil
}
