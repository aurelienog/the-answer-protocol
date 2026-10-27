package server

import (
	"log"

	"the-answer-protocol/internal/events"
	"the-answer-protocol/internal/world"
)

// cleanupPlayer removes the player's state before publishing the departure.
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
		roomID, err := gameWorld.GetPlayerRoom(session.PlayerName)
		if err != nil {
			log.Printf("Failed to get room for departing player %q: %v", session.PlayerName, err)
		} else {
			players, err := gameWorld.GetPlayersInRoom(roomID)
			if err != nil {
				log.Printf("Failed to get players in room %q: %v", roomID, err)
			} else if eventPublisher != nil {
				eventPublisher.Publish(world.PlayerLeft{
					PlayerName:  session.PlayerName,
					RoomID:      roomID,
					RoomPlayers: players,
				})
			}
		}
	}

	if registered {
		registry.Remove(sessionID)
	}

	if authenticated {
		gameWorld.RemovePlayer(session.PlayerName)
	}
}
