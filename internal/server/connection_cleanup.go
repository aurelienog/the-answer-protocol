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
	if authenticated {
		playerLeft, uniqueItemsReturned, combatEvents, statsEvent, err :=
			gameWorld.RemovePlayer(session.PlayerName)

		if err != nil {
			log.Printf(
				"Failed to remove departing player %q: %v",
				session.PlayerName,
				err,
			)
		} else {
			// Remove the session before broadcasting the updated player count.
			if registered {
				registry.Remove(sessionID)
			}

			if eventPublisher != nil {
				if playerEnterBroadcast {
					if len(uniqueItemsReturned.ItemIDs) > 0 {
						eventPublisher.Publish(uniqueItemsReturned)
					}

					eventPublisher.Publish(playerLeft)
				}

				// Publish the updated player count regardless of enterBroadcast.
				if statsEvent != nil {
					eventPublisher.Publish(*statsEvent)
				}

				// Publish all combat turn changes, NPC actions, and combat end events.
				for _, combatEvent := range combatEvents {
					if combatEvent != nil {
						eventPublisher.Publish(combatEvent)
					}
				}
			}
		}
	}

	if registered {
		registry.Remove(sessionID)
	}
}