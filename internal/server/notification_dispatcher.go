package server

import (
	"log"

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
				"Failed to queue presence notification for %q: %v",
				playerName,
				err,
			)
		}
	}
}
