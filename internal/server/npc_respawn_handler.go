package server

import (
	"log/slog"

	"the-answer-protocol/internal/events"
	"the-answer-protocol/internal/world"
)

// NewNPCRespawnHandler restores an NPC and publishes its respawn event.
func NewNPCRespawnHandler(
	gameWorld *world.World,
	publisher events.Publisher,
	logger *slog.Logger,
) func(world.NPCRespawnScheduled) {
	if logger == nil {
		panic("NPC respawn handler requires a logger")
	}
	return func(scheduled world.NPCRespawnScheduled) {
		respawned, err := gameWorld.RespawnNPC(
			scheduled.NPCID,
			scheduled.RespawnAt,
		)
		if err != nil {
			logger.Error("world.npc_respawn_failed",
				slog.String("npc", scheduled.NPCID),
				slog.String("error", err.Error()),
			)
			return
		}
		if respawned != nil && publisher != nil {
			publisher.Publish(*respawned)
		}
	}
}
