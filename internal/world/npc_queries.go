package world

import (
	"fmt"
	"strings"
)

// cloneNPCDialogue returns an independent copy of the dialogue.
func cloneNPCDialogue(dialogue NPCDialogue) NPCDialogue {
	return NPCDialogue{
		Default:   cloneSlice(dialogue.Default),
		Available: cloneSlice(dialogue.Available),
		Active:    cloneSlice(dialogue.Active),
		Completed: cloneSlice(dialogue.Completed),
		Locked:    cloneSlice(dialogue.Locked),
	}
}

// cloneNPC returns an independent copy of an NPC and its dialogue data.
func cloneNPC(npc NPC) NPC {
	npc.Dialogue = cloneNPCDialogue(npc.Dialogue)

	if npc.QuestDialogues != nil {
		dialogues := make(map[string]NPCDialogue, len(npc.QuestDialogues))
		for questID, dialogue := range npc.QuestDialogues {
			dialogues[questID] = cloneNPCDialogue(dialogue)
		}
		npc.QuestDialogues = dialogues
	}

	return npc
}

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

		npcs = append(npcs, cloneNPC(npc))
	}

	return npcs, nil
}

// TalkToNPC returns an NPC conversation for a player in the same room.
func (w *World) TalkToNPC(
	playerName, npcID string,
) (NPCConversation, error) {
	if w == nil {
		return NPCConversation{}, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	playerName = strings.TrimSpace(playerName)
	npcID = strings.TrimSpace(npcID)

	if playerName == "" || npcID == "" {
		return NPCConversation{}, fmt.Errorf(
			"%w: player name and NPC identifier cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

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

	resolvedNPCID, err := w.resolveNPCIDInRoomLocked(room, npcID)
	if err != nil {
		return NPCConversation{}, err
	}

	npcID = resolvedNPCID
	npc := w.npcs[npcID]

	conversation := NPCConversation{
		NPC: NPCSummary{
			ID:      npc.ID,
			Name:    npc.Name,
			Role:    npc.Role,
			Hostile: npc.Hostile,
			HP:      npc.HP,
			MaxHP:   npc.MaxHP,
			Damage:  npc.Damage,
		},
		Dialogue: cloneSlice(npc.Dialogue.Default),
	}

	// NPCs without quests use their default dialogue.
	if npc.QuestID == "" {
		return conversation, nil
	}

	// Start with the first quest in this NPC's sequential quest chain.
	questID := npc.QuestID
	visited := make(map[string]bool)

	for {
		if visited[questID] {
			return NPCConversation{}, fmt.Errorf(
				"%w: quest chain for NPC %q contains a cycle at %q",
				ErrInvalidWorldState,
				npcID,
				questID,
			)
		}
		visited[questID] = true

		quest, exists := w.quests[questID]
		if !exists {
			return NPCConversation{}, fmt.Errorf(
				"%w: NPC %q references nonexistent quest %q",
				ErrInvalidWorldState,
				npcID,
				questID,
			)
		}

		if quest.GiverNPCID != npcID {
			return NPCConversation{}, fmt.Errorf(
				"%w: quest %q is not assigned to NPC %q",
				ErrInvalidWorldState,
				questID,
				npcID,
			)
		}

		playerQuests := w.playerQuests[playerName]
		progress, hasProgress := playerQuests[quest.ID]

		var status QuestStatus

		if hasProgress {
			status = progress.Status

			// Move to the next quest only after this one is completed.
			if status == QuestStatusCompleted && quest.NextQuestID != "" {
				questID = quest.NextQuestID
				continue
			}
		} else {
			// Check whether this quest's prerequisites are satisfied.
			canAccept, _ := w.canAcceptQuestLocked(playerName, quest.ID)

			if !canAccept {
				status = QuestStatusLocked
			} else {
				// Talking to the giver discovers the quest.
				if w.playerQuests == nil {
					w.playerQuests = make(map[string]map[string]PlayerQuest)
				}

				if w.playerQuests[playerName] == nil {
					w.playerQuests[playerName] = make(map[string]PlayerQuest)
				}

				w.playerQuests[playerName][quest.ID] = PlayerQuest{
					QuestID: quest.ID,
					Status:  QuestStatusAvailable,
				}

				status = QuestStatusAvailable
			}
		}

		conversation.Quest = &QuestSummary{
			ID:          quest.ID,
			Name:        quest.Name,
			Description: quest.Description,
		}
		conversation.QuestStatus = status

		// Select the dialogue configured for this quest and status.
		questDialogue, exists := npc.QuestDialogues[quest.ID]
		if !exists {
			questDialogue = NPCDialogue{}
		}

		conversation.Dialogue = dialogueForQuestStatus(
			questDialogue,
			status,
		)

		// Fall back to the NPC's general dialogue if no quest-specific
		// lines are configured for this status.
		if len(conversation.Dialogue) == 0 {
			conversation.Dialogue = cloneSlice(npc.Dialogue.Default)
		}

		return conversation, nil
	}
}

// ResolveNPCID resolves an NPC ID or name among the NPCs in the player's room.
func (w *World) ResolveNPCID(playerName, identifier string) (string, error) {
	if w == nil {
		return "", fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	playerName = strings.TrimSpace(playerName)
	identifier = strings.TrimSpace(identifier)

	if playerName == "" || identifier == "" {
		return "", fmt.Errorf(
			"%w: player name and NPC identifier cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	roomID, exists := w.playerRooms[playerName]
	if !exists {
		return "", fmt.Errorf(
			"%w: %q",
			ErrPlayerNotFound,
			playerName,
		)
	}

	room, exists := w.rooms[roomID]
	if !exists {
		return "", fmt.Errorf(
			"%w: player %q references nonexistent room %q",
			ErrInvalidWorldState,
			playerName,
			roomID,
		)
	}

	return w.resolveNPCIDInRoomLocked(room, identifier)
}

// resolveNPCIDInRoomLocked resolves an NPC ID or name within a room.
// The caller must already hold the world mutex.
func (w *World) resolveNPCIDInRoomLocked(
	room Room,
	identifier string,
) (string, error) {
	identifier = strings.TrimSpace(identifier)

	if identifier == "" {
		return "", fmt.Errorf(
			"%w: NPC identifier cannot be empty",
			ErrInvalidArgument,
		)
	}

	var resolvedNPCID string

	for _, npcID := range room.NPCs {
		npc, exists := w.npcs[npcID]
		if !exists {
			return "", fmt.Errorf(
				"%w: room %q references nonexistent NPC %q",
				ErrInvalidWorldState,
				room.ID,
				npcID,
			)
		}

		if !strings.EqualFold(npc.ID, identifier) &&
			!strings.EqualFold(npc.Name, identifier) {
			continue
		}

		if resolvedNPCID != "" && resolvedNPCID != npc.ID {
			return "", fmt.Errorf(
				"%w: NPC identifier %q is ambiguous",
				ErrInvalidArgument,
				identifier,
			)
		}

		resolvedNPCID = npc.ID
	}

	if resolvedNPCID == "" {
		return "", fmt.Errorf(
			"%w: NPC %q is not in room %q",
			ErrNPCNotFound,
			identifier,
			room.ID,
		)
	}

	return resolvedNPCID, nil
}