package server

import (
	"log"

	"the-answer-protocol/internal/events"
	"the-answer-protocol/internal/world"
)

// cleanupPlayer removes the player's state before publishing departure events.
func cleanupPlayer(
	sessionID string,
	session *Session,
	registry *Registry,
	gameWorld *world.World,
	registered bool,
	authenticated bool,
	playerEnterBroadcast bool,
	eventPublisher events.Publisher,
) {
	if authenticated && playerEnterBroadcast {
		playerLeft, uniqueItemsReturned, turnEvent, err := gameWorld.RemovePlayer(session.PlayerName)
		if err != nil {
			log.Printf(
				"Failed to remove departing player %q: %v",
				session.PlayerName,
				err,
			)
		} else {
			// The world state has already been updated before publishing.
			if eventPublisher != nil {
				
				if len(uniqueItemsReturned.ItemIDs) > 0 {
					eventPublisher.Publish(uniqueItemsReturned)
				}

				eventPublisher.Publish(playerLeft)

				if turnEvent != nil {
					eventPublisher.Publish(*turnEvent)
				}
			}
		}
	} else if authenticated {
		// The player was authenticated but their entry was not broadcast.
		// Remove their state without publishing departure events.
		_, _, turnEvent, err := gameWorld.RemovePlayer(session.PlayerName)
		if err != nil {
			log.Printf(
				"Failed to remove departing player %q: %v",
				session.PlayerName,
				err,
			)
		}
		if turnEvent != nil && eventPublisher != nil {
			eventPublisher.Publish(*turnEvent)
		}
	}

	if registered {
		registry.Remove(sessionID)
	}
}
