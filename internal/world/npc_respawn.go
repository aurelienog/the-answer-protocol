package world

import (
	"fmt"
	"strings"
	"time"
)

// refreshNPCRespawnLocked restores an NPC when its respawn time has elapsed.
// The caller must hold w.mu.Lock().
func (w *World) refreshNPCRespawnLocked(npcID string) (*NPCRespawned, bool) {
	npc, exists := w.npcs[npcID]
	if !exists || npc.CombatStats.HP > 0 || npc.RespawnAt.IsZero() {
		return nil, false
	}

	if time.Now().Before(npc.RespawnAt) {
		return nil, false
	}

	npc.CombatStats.HP = npc.CombatStats.MaxHP
	npc.RespawnAt = time.Time{}

	w.npcs[npcID] = npc
	roomID := ""
	for id, room := range w.rooms {
		for _, roomNPCID := range room.NPCs {
			if roomNPCID == npcID {
				roomID = id
				return &NPCRespawned{
					NPCID:       npcID,
					NPCName:     npc.Name,
					RoomID:      roomID,
					RoomPlayers: w.getPlayersInRoomLocked(roomID),
				}, true
			}
		}
	}
	return &NPCRespawned{
		NPCID:   npcID,
		NPCName: npc.Name,
		RoomID:  roomID,
	}, true
}

// RespawnNPC restores an NPC if its matching respawn deadline has elapsed.
func (w *World) RespawnNPC(
	npcID string,
	expectedRespawnAt time.Time,
) (*NPCRespawned, error) {
	if w == nil {
		return nil, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	npcID = strings.TrimSpace(npcID)
	if npcID == "" || expectedRespawnAt.IsZero() {
		return nil, fmt.Errorf(
			"%w: NPC ID and respawn deadline are required",
			ErrInvalidArgument,
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	npc, exists := w.npcs[npcID]
	if !exists {
		return nil, fmt.Errorf("%w: %q", ErrNPCNotFound, npcID)
	}
	if npc.CombatStats.HP > 0 ||
		npc.RespawnAt.IsZero() ||
		!npc.RespawnAt.Equal(expectedRespawnAt) {
		return nil, nil
	}
	if time.Now().Before(npc.RespawnAt) {
		return nil, fmt.Errorf("%w: %q", ErrNPCRespawning, npcID)
	}

	event, respawned := w.refreshNPCRespawnLocked(npcID)
	if !respawned {
		return nil, nil
	}
	return event, nil
}
