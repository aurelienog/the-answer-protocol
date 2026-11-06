package world

import (
	"fmt"
	"sort"
)

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

	if err := w.validateRoomExitRelations(); err != nil {
		return err
	}

		if err := w.validateInitialRoomRelation(); err != nil {
		return err
	}

	if err := w.validateRoomConnectivity(); err != nil {
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

// validateRoomExitRelations checks that all room exits have a matching reverse exit.
func (w *World) validateRoomExitRelations() error {
	reverseDirections := map[string]string{
		"north": "south",
		"south": "north",
		"east":  "west",
		"west":  "east",
	}

	for roomID, room := range w.rooms {
		for direction, destinationID := range room.Exits {
			destination, exists := w.rooms[destinationID]
			if !exists {
				// The individual room validator already checks this reference.
				continue
			}

			reverseDirection := reverseDirections[direction]
			if reverseDestination, exists := destination.Exits[reverseDirection]; !exists ||
				reverseDestination != roomID {
				return fmt.Errorf(
					"room %q has exit %q to room %q, but the reverse exit %q is missing or incorrect",
					roomID,
					direction,
					destinationID,
					reverseDirection,
				)
			}
		}
	}

	return nil
}

// validateInitialRoomRelation checks that the initial room exists.
func (w *World) validateInitialRoomRelation() error {
	if _, exists := w.rooms[InitialRoomID]; !exists {
		return fmt.Errorf(
			"initial room %q does not exist",
			InitialRoomID,
		)
	}

	return nil
}

// validateRoomConnectivity checks that every room is reachable from the initial room.
func (w *World) validateRoomConnectivity() error {
	visited := make(map[string]struct{})
	queue := []string{InitialRoomID}

	visited[InitialRoomID] = struct{}{}

	for len(queue) > 0 {
		roomID := queue[0]
		queue = queue[1:]

		room := w.rooms[roomID]

		for _, destinationID := range room.Exits {
			if _, exists := visited[destinationID]; exists {
				continue
			}

			if _, exists := w.rooms[destinationID]; !exists {
				continue
			}

			visited[destinationID] = struct{}{}
			queue = append(queue, destinationID)
		}
	}

	var unreachableRooms []string

	for roomID := range w.rooms {
		if _, exists := visited[roomID]; !exists {
			unreachableRooms = append(unreachableRooms, roomID)
		}
	}

	if len(unreachableRooms) > 0 {
		sort.Strings(unreachableRooms)

		return fmt.Errorf(
			"rooms are not reachable from initial room %q: %v",
			InitialRoomID,
			unreachableRooms,
		)
	}

	return nil
}