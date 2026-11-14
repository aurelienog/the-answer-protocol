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
		playerLeft, uniqueItemsReturned, turnEvent, combatsToEnd, statsEvent, err :=
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

				// Notify players about the updated combat turn.
				if turnEvent != nil {
					eventPublisher.Publish(*turnEvent)
				}
			}

			// End combats that no longer have active players.
			for _, combatID := range combatsToEnd {
				_, event, err := gameWorld.EndCombat(combatID)
				if err != nil {
					log.Printf(
						"Failed to end combat %q after player %q disconnected: %v",
						combatID,
						session.PlayerName,
						err,
					)
					continue
				}

				if event != nil && eventPublisher != nil {
					eventPublisher.Publish(*event)
				}
			}
		}
	}

	if registered {
		registry.Remove(sessionID)
	}
}