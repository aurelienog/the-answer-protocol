package world

import "fmt"

// GetNPCsInRoom returns the NPC definitions for all NPCs in a room.
func (w *World) GetNPCsInRoom(roomID string) ([]NPC, error) {
	if w == nil {
		return nil, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	room, exists := w.rooms[roomID]
	if !exists {
		return nil, fmt.Errorf(
			"%w: room %q does not exist",
			ErrInvalidWorldState,
			roomID,
		)
	}

	npcs := make([]NPC, 0, len(room.NPCs))

	for _, npcID := range room.NPCs {
		npc, exists := w.npcs[npcID]
		if !exists {
			return nil, fmt.Errorf(
				"%w: room %q references nonexistent NPC %q",
				ErrInvalidWorldState,
				roomID,
				npcID,
			)
		}

		// Return a copy of the dialogue slice to protect world state.
		npc.Dialogue = cloneSlice(npc.Dialogue)

		npcs = append(npcs, npc)
	}

	return npcs, nil
}

// TalkToNPC returns an NPC conversation for a player in the same room.
func (w *World) TalkToNPC(playerName, npcID string) (NPCConversation, error) {
	if w == nil {
		return NPCConversation{}, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	if playerName == "" || npcID == "" {
		return NPCConversation{}, fmt.Errorf(
			"%w: player name and NPC ID cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	roomID, exists := w.playerRooms[playerName]
	if !exists {
		return NPCConversation{}, fmt.Errorf(
			"%w: %q",
			ErrPlayerNotFound,
			playerName,
		)
	}

	room, exists := w.rooms[roomID]
	if !exists {
		return NPCConversation{}, fmt.Errorf(
			"%w: player %q references nonexistent room %q",
			ErrInvalidWorldState,
			playerName,
			roomID,
		)
	}

	npc, exists := w.npcs[npcID]
	if !exists {
		return NPCConversation{}, fmt.Errorf(
			"%w: %q",
			ErrNPCNotFound,
			npcID,
		)
	}

	npcPresent := false
	for _, roomNPCID := range room.NPCs {
		if roomNPCID == npcID {
			npcPresent = true
			break
		}
	}

	if !npcPresent {
		return NPCConversation{}, fmt.Errorf(
			"%w: NPC %q is not in the player's room",
			ErrNPCNotFound,
			npcID,
		)
	}

	conversation := NPCConversation{
		NPC:      npc,
		Dialogue: cloneSlice(npc.Dialogue),
	}

	// Resolve the mission associated with this NPC, if any.
	if npc.QuestID == "" {
		return conversation, nil
	}

	quest, exists := w.quests[npc.QuestID]
	if !exists {
		return NPCConversation{}, fmt.Errorf(
			"%w: NPC %q references nonexistent quest %q",
			ErrInvalidWorldState,
			npcID,
			npc.QuestID,
		)
	}

	quest.RewardItemIDs = cloneSlice(quest.RewardItemIDs)
	conversation.Quest = &quest

	if playerQuests, exists := w.playerQuests[playerName]; exists {
		if progress, exists := playerQuests[quest.ID]; exists {
			conversation.QuestStatus = progress.Status
			return conversation, nil
		}
	}

	canAccept, _ := w.canAcceptQuestLocked(playerName, quest.ID)
	if canAccept {
		conversation.QuestStatus = QuestStatusAvailable
	} else {
		conversation.QuestStatus = QuestStatusLocked
	}

	return conversation, nil
}