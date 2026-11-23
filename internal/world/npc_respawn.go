package world

import "time"

// refreshNPCRespawnLocked restores an NPC when its respawn time has elapsed.
// The caller must hold w.mu.Lock().
func (w *World) refreshNPCRespawnLocked(npcID string) (*NPCRespawned, bool) {
	npc, exists := w.npcs[npcID]
	if !exists || npc.CombatStats.HP > 0 || npc.RespawnAt <= 0 {
		return nil, false
	}

	if time.Now().Unix() < npc.RespawnAt {
		return nil, false
	}

	npc.CombatStats.HP = npc.CombatStats.MaxHP
	npc.RespawnAt = 0

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
