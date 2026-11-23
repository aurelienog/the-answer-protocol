package logger

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"the-answer-protocol/internal/world"
)

func New() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	})

	return slog.New(handler)
}

// AuditLogger records real world changes published to the event bus.
type AuditLogger struct {
	log *slog.Logger
}

func NewAuditLogger(log *slog.Logger) *AuditLogger {
	if log == nil {
		panic("audit logger requires a logger")
	}
	return &AuditLogger{log: log}
}

func (audit *AuditLogger) Handle(event any) {
	if audit == nil || audit.log == nil || event == nil {
		return
	}

	switch value := event.(type) {
	case world.PlayerMoved:
		audit.log.Info("world.player_moved", slog.String("player", value.PlayerName), slog.String("from_room", value.PreviousRoomID), slog.String("to_room", value.NewRoomID))
	case *world.PlayerMoved:
		if value != nil {
			audit.log.Info("world.player_moved", slog.String("player", value.PlayerName), slog.String("from_room", value.PreviousRoomID), slog.String("to_room", value.NewRoomID))
		}
	case world.PlayerEntered:
		audit.log.Info("world.player_entered", slog.String("player", value.PlayerName), slog.String("room", value.RoomID))
	case world.PlayerLeft:
		audit.log.Info("world.player_left", slog.String("player", value.PlayerName), slog.String("room", value.RoomID))
	case world.ItemDropped:
		audit.log.Info("world.item_dropped", slog.String("player", value.PlayerName), slog.String("item", value.ItemID), slog.String("room", value.RoomID))
	case world.UniqueItemsReturned:
		audit.log.Info("world.unique_items_returned", slog.String("player", value.PlayerName), slog.String("room", value.RoomID), slog.String("items", fmt.Sprint(value.ItemIDs)))
	case world.ChatMessage:
		audit.log.Info("world.chat_message", slog.String("player", value.PlayerName), slog.String("scope", value.Scope), slog.Int("recipients", len(value.Recipients)))
	case world.GroupInvite:
		audit.log.Info("world.group_invite", slog.String("inviter", value.Inviter), slog.String("invitee", value.Invitee), slog.String("group", value.GroupName))
	case world.GroupJoin:
		audit.log.Info("world.group_join", slog.String("player", value.PlayerName), slog.String("group", value.GroupName))
	case world.GroupLeave:
		audit.log.Info("world.group_leave", slog.String("player", value.PlayerName), slog.String("group", value.GroupName))
	case world.CombatStarted:
		audit.log.Info("world.combat_started", slog.String("combat", value.CombatID), slog.String("room", value.RoomID), slog.Int("participants", len(value.Participants)))
	case world.CombatAttack:
		audit.log.Info("world.combat_attack", slog.String("combat", value.CombatID), slog.String("attacker", value.AttackerName), slog.String("target", value.TargetName))
	case world.CombatDamage:
		audit.log.Info("world.combat_damage", slog.String("combat", value.CombatID), slog.String("attacker", value.AttackerName), slog.String("target", value.TargetName), slog.Int("damage", value.Damage), slog.Int("remaining_hp", value.RemainingHP))
	case world.ParticipantDied:
		audit.log.Info("world.participant_died", slog.String("combat", value.CombatID), slog.String("participant", value.ParticipantName), slog.String("killer", value.KillerID))
	case world.PlayerFled:
		audit.log.Info("world.player_fled", slog.String("combat", value.CombatID), slog.String("player", value.PlayerName), slog.Int("remaining_hp", value.RemainingHP))
	case world.CombatTurnChanged:
		audit.log.Info("world.combat_turn_changed", slog.String("combat", value.CombatID), slog.String("participant", value.ParticipantName), slog.Int("round", value.Round))
	case world.CombatPlayerJoined:
		audit.log.Info("world.combat_player_joined", slog.String("combat", value.CombatID), slog.String("player", value.PlayerName))
	case world.CombatHelpRequested:
		audit.log.Info("world.combat_help_requested", slog.String("combat", value.CombatID), slog.String("player", value.PlayerName), slog.String("room", value.RoomID))
	case world.CombatEnded:
		audit.log.Info("world.combat_ended", slog.String("combat", value.CombatID), slog.Int("rewards", len(value.Rewards)))
	case *world.CombatEnded:
		if value != nil {
			audit.log.Info("world.combat_ended", slog.String("combat", value.CombatID), slog.Int("rewards", len(value.Rewards)))
		}
	case world.ItemUsed:
		audit.log.Info("world.item_used", slog.String("player", value.PlayerName), slog.String("item", value.ItemID))
	case world.PlayerCountChanged:
		audit.log.Info("world.player_count_changed", slog.Int("players", value.Count), slog.String("revision", strconv.FormatUint(value.Revision, 10)))
	case world.PlayerRespawned:
		audit.log.Info("world.player_respawned", slog.String("player", value.PlayerName), slog.String("room", value.RoomID), slog.Int("hp", value.HP))
	default:
		audit.log.Info("world.event", slog.String("event_type", fmt.Sprintf("%T", event)))
	}
}
