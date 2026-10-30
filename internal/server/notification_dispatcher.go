package server

import (
	"log"
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
			protocol.EVT("ROOM_PRESENCE_ENTER", event.PlayerName),
		)

	case world.PlayerLeft:
		dispatcher.notifyPlayers(
			event.RoomPlayers,
			event.PlayerName,
			protocol.EVT("ROOM_PRESENCE_LEAVE", event.PlayerName),
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

	}
}

func (dispatcher *NotificationDispatcher) handlePlayerMoved(event world.PlayerMoved) {
	// Notify the players who occupied the previous room before the move.
	dispatcher.notifyPlayers(
		event.PreviousRoomPlayers,
		event.PlayerName,
		protocol.EVT("ROOM_PRESENCE_LEAVE", event.PlayerName),
	)

	// Notify the players who occupied the destination room before the move.
	dispatcher.notifyPlayers(
		event.NewRoomPlayers,
		event.PlayerName,
		protocol.EVT("ROOM_PRESENCE_ENTER", event.PlayerName),
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