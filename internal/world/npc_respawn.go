package world

import "time"

// refreshNPCRespawnLocked restores an NPC when its respawn time has elapsed.
// The caller must hold w.mu.Lock().
func (w *World) refreshNPCRespawnLocked(npcID string) bool {
	npc, exists := w.npcs[npcID]
	if !exists || npc.CombatStats.HP > 0 || npc.RespawnAt <= 0 {
		return false
	}

	if time.Now().Unix() < npc.RespawnAt {
		return false
	}

	npc.CombatStats.HP = npc.CombatStats.MaxHP
	npc.RespawnAt = 0

	w.npcs[npcID] = npc

	return true
}