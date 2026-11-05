package server

import (
	"log"
	"strconv"
	"strings"
	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

// NotificationDispatcher translates internal game events into TAP notifications.
type NotificationDispatcher struct {
	registry *Registry
}

// NewNotificationDispatcher creates a notification dispatcher.
func NewNotificationDispatcher(registry *Registry) *NotificationDispatcher {
	return &NotificationDispatcher{
		registry: registry,
	}
}

// Handle processes an internal game event.
func (dispatcher *NotificationDispatcher) Handle(event any) {
	if dispatcher == nil || dispatcher.registry == nil {
		return
	}

	switch event := event.(type) {
	case world.PlayerMoved:
		dispatcher.handlePlayerMoved(event)

	case world.PlayerEntered:
		dispatcher.notifyPlayers(
			event.RoomPlayers,
			event.PlayerName,
			protocol.EVT("ROOM", "PRESENCE ENTER "+event.PlayerName),
		)

	case world.PlayerLeft:
		dispatcher.notifyPlayers(
			event.RoomPlayers,
			event.PlayerName,
			protocol.EVT("ROOM", "PRESENCE LEAVE "+event.PlayerName),
		)

	case world.ItemDropped:
		dispatcher.handleItemDropped(event)

	case world.UniqueItemsReturned:
		dispatcher.handleUniqueItemsReturned(event)

	case world.GroupInvite:
		dispatcher.handleGroupInvite(event)

	case world.GroupJoin:
		dispatcher.handleGroupJoin(event)

	case world.GroupLeave:
		dispatcher.handleGroupLeave(event)

	case world.ChatMessage:
		dispatcher.handleChatMessage(event)

	case world.ParticipantDied:
		dispatcher.handleParticipantDied(event)

	case world.CombatStarted:
		dispatcher.handleCombatStarted(event)

	case world.CombatAttack:
		dispatcher.handleCombatAttack(event)

	case world.CombatDamage:
		dispatcher.handleCombatDamage(event)

	case world.CombatHelpRequested:
		dispatcher.handleCombatHelpRequested(event)

	case world.PlayerFled:
		dispatcher.handlePlayerFled(event)

	case world.CombatTurnChanged:
		dispatcher.handleCombatTurnChanged(event)

	case world.CombatPlayerJoined:
		dispatcher.handleCombatPlayerJoined(event)

	case world.ItemUsed:
		dispatcher.handleItemUsed(event)
	}

}

func (dispatcher *NotificationDispatcher) handlePlayerMoved(event world.PlayerMoved) {
	// Notify the players who occupied the previous room before the move.
	dispatcher.notifyPlayers(
		event.PreviousRoomPlayers,
		event.PlayerName,
		protocol.EVT("ROOM", "PRESENCE LEAVE "+event.PlayerName),
	)

	// Notify the players who occupied the destination room before the move.
	dispatcher.notifyPlayers(
		event.NewRoomPlayers,
		event.PlayerName,
		protocol.EVT("ROOM", "PRESENCE ENTER "+event.PlayerName),
	)
}

func (dispatcher *NotificationDispatcher) handleItemDropped(event world.ItemDropped) {
	dispatcher.notifyPlayers(
		event.RoomPlayers,
		event.PlayerName,
		protocol.EVT("ITEM_DROPPED", event.ItemID),
	)
}

func (dispatcher *NotificationDispatcher) handleUniqueItemsReturned(event world.UniqueItemsReturned) {
	if len(event.ItemIDs) == 0 {
		return
	}

	// ItemIDs currently contains the concrete instance IDs returned to the room.
	// Send one global notification describing the return.
	itemIDs := strings.Join(event.ItemIDs, ",")
	dispatcher.registry.Broadcast(
		protocol.EVT("UNIQUE_ITEMS_RETURNED", itemIDs),
	)
}

func (dispatcher *NotificationDispatcher) notifyPlayers(
	playerNames []string,
	movingPlayer string,
	message protocol.Message,
) {
	for _, playerName := range playerNames {
		if playerName == movingPlayer {
			continue
		}

		session, exists := dispatcher.registry.GetByName(playerName)
		if !exists {
			continue
		}

		if err := dispatcher.registry.Queue(session.ID, message); err != nil {
			log.Printf(
				"Failed to queue notification for %q: %v",
				playerName,
				err,
			)
		}
	}
}

func (dispatcher *NotificationDispatcher) handleGroupInvite(event world.GroupInvite) {
	session, exists := dispatcher.registry.GetByName(event.Invitee)
	if !exists {
		return
	}

	if err := dispatcher.registry.Queue(
		session.ID,
		protocol.EVT(
			"GROUP",
			"INVITE player="+event.Inviter+" group="+event.GroupName,
		),
	); err != nil {
		log.Printf(
			"Failed to queue group invitation notification for %q: %v",
			event.Invitee,
			err,
		)
	}
}

func (dispatcher *NotificationDispatcher) handleGroupJoin(event world.GroupJoin) {
	dispatcher.notifyPlayers(
		event.Members,
		event.PlayerName,
		protocol.EVT("GROUP", "JOIN player="+event.PlayerName),
	)
}

func (dispatcher *NotificationDispatcher) handleGroupLeave(event world.GroupLeave) {
	dispatcher.notifyPlayers(
		event.Members,
		event.PlayerName,
		protocol.EVT("GROUP", "LEAVE player="+event.PlayerName),
	)
}

func (dispatcher *NotificationDispatcher) handleChatMessage(event world.ChatMessage) {
	data := "CHAT player=" + event.PlayerName + " message=" + event.Message
	notification := protocol.EVT(event.Scope, data)

	for _, playerName := range event.Recipients {
		session, exists := dispatcher.registry.GetByName(playerName)
		if !exists {
			continue
		}

		if err := dispatcher.registry.Queue(session.ID, notification); err != nil {
			log.Printf(
				"Failed to queue chat message for %q: %v",
				playerName,
				err,
			)
		}
	}
}

func (dispatcher *NotificationDispatcher) handleParticipantDied(
	event world.ParticipantDied,
) {
	participantType := "NPC"

	if event.IsPlayer {
		participantType = "PLAYER"
	}

	notification := protocol.EVT(
		"COMBAT",
		participantType+"_DIED participant="+event.ParticipantName+
			" killer="+event.KillerID,
	)

	dispatcher.notifyPlayers(
		event.Recipients,
		"",
		notification,
	)
}

func (dispatcher *NotificationDispatcher) handleCombatStarted(
	event world.CombatStarted,
) {
	participants := strings.Join(event.Participants, ",")

	notification := protocol.EVT(
		"COMBAT",
		"STARTED combat="+event.CombatID+
			" room="+event.RoomID+
			" participants="+participants,
	)

	dispatcher.notifyPlayers(
		event.Recipients,
		"",
		notification,
	)
}

func (dispatcher *NotificationDispatcher) handleCombatAttack(
	event world.CombatAttack,
) {
	attackerType := "NPC"

	if event.IsPlayer {
		attackerType = "PLAYER"
	}

	notification := protocol.EVT(
		"COMBAT",
		"ATTACK attacker_type="+attackerType+
			" attacker="+event.AttackerName+
			" target="+event.TargetName,
	)

	dispatcher.notifyPlayers(
		event.Recipients,
		"",
		notification,
	)
}

func (dispatcher *NotificationDispatcher) handleCombatDamage(
	event world.CombatDamage,
) {
	notification := protocol.EVT(
		"COMBAT",
		"DAMAGE attacker="+event.AttackerName+
			" target="+event.TargetName+
			" amount="+strconv.Itoa(event.Damage)+
			" remaining_hp="+strconv.Itoa(event.RemainingHP),
	)

	dispatcher.notifyPlayers(event.Recipients, "", notification)
}

func (dispatcher *NotificationDispatcher) handlePlayerFled(
	event world.PlayerFled,
) {
	notification := protocol.EVT(
		"COMBAT",
		"PLAYER_FLED player="+event.PlayerName+
			" damage="+strconv.Itoa(event.Damage)+
			" remaining_hp="+strconv.Itoa(event.RemainingHP),
	)

	dispatcher.notifyPlayers(
		event.Recipients,
		event.PlayerName,
		notification,
	)
}

func (dispatcher *NotificationDispatcher) handleCombatTurnChanged(
	event world.CombatTurnChanged,
) {
	participantType := "NPC"

	if event.IsPlayer {
		participantType = "PLAYER"
	}

	notification := protocol.EVT(
		"COMBAT",
		"TURN participant_type="+participantType+
			" participant="+event.ParticipantName+
			" round="+strconv.Itoa(event.Round)+
			" actions="+strconv.Itoa(event.AvailableActions),
	)

	dispatcher.notifyPlayers(
		event.Recipients,
		"",
		notification,
	)
}

func (dispatcher *NotificationDispatcher) handleItemUsed(
	event world.ItemUsed,
) {
	session, exists := dispatcher.registry.GetByName(event.PlayerName)
	if !exists {
		return
	}

	data := "player=" + event.PlayerName +
		" item=" + event.ItemID

	if hp, exists := event.StatChanges[world.CombatStatHP]; exists {
		data += " hp=" + strconv.Itoa(hp)
	}

	if err := dispatcher.registry.Queue(
		session.ID,
		protocol.EVT("ITEM_USED", data),
	); err != nil {
		log.Printf(
			"Failed to queue item used notification for %q: %v",
			event.PlayerName,
			err,
		)
	}
}

func (dispatcher *NotificationDispatcher) handleCombatHelpRequested(
	event world.CombatHelpRequested,
) {
	notification := protocol.EVT(
		"COMBAT",
		"HELP_REQUESTED player="+event.PlayerName+
			" combat="+event.CombatID+
			" room="+event.RoomID,
	)

	dispatcher.notifyPlayers(
		event.Recipients,
		event.PlayerName,
		notification,
	)
}

func (dispatcher *NotificationDispatcher) handleCombatPlayerJoined(
	event world.CombatPlayerJoined,
) {
	notification := protocol.EVT(
		"COMBAT",
		"PLAYER_JOINED player="+event.PlayerName+
			" combat="+event.CombatID,
	)

	dispatcher.notifyPlayers(
		event.Recipients,
		"",
		notification,
	)
}
