package server

import (
	"log/slog"

	"the-answer-protocol/internal/world"
)

// AuditLogger subscribes to the internal event bus and logs world, chat,
// group, and combat state transitions as structured JSON entries.
type AuditLogger struct {
	log *slog.Logger
}

// NewAuditLogger creates an event-bus audit logger.
func NewAuditLogger(log *slog.Logger) *AuditLogger {
	return &AuditLogger{log: log}
}

// Handle logs an internal domain event.
func (a *AuditLogger) Handle(event any) {
	if a == nil || a.log == nil {
		return
	}

	switch e := event.(type) {
	case world.PlayerEntered:
		a.log.Info("world.player_entered",
			slog.String("player", e.PlayerName),
			slog.String("room_id", e.RoomID),
		)

	case world.PlayerLeft:
		a.log.Info("world.player_left",
			slog.String("player", e.PlayerName),
			slog.String("room_id", e.RoomID),
		)

	case world.PlayerMoved:
		a.log.Info("world.player_moved",
			slog.String("player", e.PlayerName),
			slog.String("from_room", e.PreviousRoomID),
			slog.String("to_room", e.NewRoomID),
		)

	case world.ItemDropped:
		a.log.Info("world.item_dropped",
			slog.String("player", e.PlayerName),
			slog.String("item_id", e.ItemID),
			slog.String("room_id", e.RoomID),
		)

	case world.ItemUsed:
		a.log.Info("world.item_used",
			slog.String("player", e.PlayerName),
			slog.String("item_id", e.ItemID),
			slog.Any("stat_changes", e.StatChanges),
		)

	case world.UniqueItemsReturned:
		a.log.Info("world.unique_items_returned",
			slog.String("player", e.PlayerName),
			slog.String("room_id", e.RoomID),
			slog.Any("item_instances", e.ItemIDs),
		)

	case world.GroupInvite:
		a.log.Info("group.invite",
			slog.String("group_id", e.GroupID),
			slog.String("group_name", e.GroupName),
			slog.String("inviter", e.Inviter),
			slog.String("invitee", e.Invitee),
		)

	case world.GroupJoin:
		a.log.Info("group.join",
			slog.String("group_id", e.GroupID),
			slog.String("group_name", e.GroupName),
			slog.String("player", e.PlayerName),
		)

	case world.GroupLeave:
		a.log.Info("group.leave",
			slog.String("group_id", e.GroupID),
			slog.String("group_name", e.GroupName),
			slog.String("player", e.PlayerName),
		)

	case world.ChatMessage:
		a.log.Info("chat.message",
			slog.String("scope", e.Scope),
			slog.String("player", e.PlayerName),
			slog.Int("recipients", len(e.Recipients)),
		)

	case world.CombatStarted:
		participants := make([]string, 0, len(e.Participants))
		for _, p := range e.Participants {
			participants = append(participants, p.ID)
		}
		a.log.Info("combat.started",
			slog.String("combat_id", e.CombatID),
			slog.String("room_id", e.RoomID),
			slog.Any("participants", participants),
		)

	case world.CombatAttack:
		a.log.Info("combat.attack",
			slog.String("combat_id", e.CombatID),
			slog.String("attacker", e.AttackerID),
			slog.String("target", e.TargetID),
			slog.Bool("attacker_is_player", e.IsPlayer),
		)

	case world.CombatDamage:
		a.log.Info("combat.damage",
			slog.String("combat_id", e.CombatID),
			slog.String("attacker", e.AttackerID),
			slog.String("target", e.TargetID),
			slog.Int("damage", e.Damage),
			slog.Int("remaining_hp", e.RemainingHP),
		)

	case world.ParticipantDied:
		a.log.Info("combat.participant_died",
			slog.String("combat_id", e.CombatID),
			slog.String("participant", e.ParticipantID),
			slog.Bool("is_player", e.IsPlayer),
			slog.String("killer", e.KillerID),
		)

	case world.PlayerFled:
		a.log.Info("combat.player_fled",
			slog.String("combat_id", e.CombatID),
			slog.String("player", e.PlayerName),
			slog.Int("damage", e.Damage),
			slog.Int("remaining_hp", e.RemainingHP),
		)

	case world.CombatPlayerJoined:
		a.log.Info("combat.player_joined",
			slog.String("combat_id", e.CombatID),
			slog.String("player", e.PlayerName),
		)

	case world.CombatHelpRequested:
		a.log.Info("combat.help_requested",
			slog.String("combat_id", e.CombatID),
			slog.String("player", e.PlayerName),
			slog.String("room_id", e.RoomID),
		)

	case world.CombatEnded:
		a.log.Info("combat.ended",
			slog.String("combat_id", e.CombatID),
			slog.Int("reward_batches", len(e.Rewards)),
		)

	case world.PlayerRespawned:
		a.log.Info("combat.player_respawned",
			slog.String("combat_id", e.CombatID),
			slog.String("player", e.PlayerName),
			slog.String("room_id", e.RoomID),
			slog.Int("hp", e.HP),
			slog.Int("max_hp", e.MaxHP),
		)
	}
}