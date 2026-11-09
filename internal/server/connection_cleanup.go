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
		playerLeft, uniqueItemsReturned, turnEvent, combatsToEnd, err :=
			gameWorld.RemovePlayer(session.PlayerName)

		if err != nil {
			log.Printf(
				"Failed to remove departing player %q: %v",
				session.PlayerName,
				err,
			)
		} else {
				if registered {
					registry.Remove(sessionID)
				}

			if playerEnterBroadcast && eventPublisher != nil {
				// The world state has already been updated before publishing.
				if len(uniqueItemsReturned.ItemIDs) > 0 {
					eventPublisher.Publish(uniqueItemsReturned)
				}

				eventPublisher.Publish(playerLeft)
			}

			if turnEvent != nil && eventPublisher != nil {
				eventPublisher.Publish(*turnEvent)
			}

			// End combats that no longer have active players.
			for _, combatID := range combatsToEnd {
				if _, err := gameWorld.EndCombat(combatID); err != nil {
					log.Printf(
						"Failed to end combat %q after player %q disconnected: %v",
						combatID,
						session.PlayerName,
						err,
					)
				} else if eventPublisher != nil {
					eventPublisher.Publish(world.CombatEnded{CombatID: combatID})
				}
			}
		}
	}

	if registered {
		registry.Remove(sessionID)
	}
}
