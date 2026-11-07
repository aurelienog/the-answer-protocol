package server

import (
	"errors"
	"log"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

// parseConnectRequest validates a CONNECT request and returns its username.
func parseConnectRequest(line string) (string, *protocol.Message) {
	request, err := protocol.ParseRequest(line)
	if err != nil {
		message := protocol.ERR(
			protocol.CodeInvalidRequest,
			"INVALID_REQUEST",
		)
		return "", &message
	}

	if request.Command != "CONNECT" {
		message := protocol.ERR(
			protocol.CodeConnectRequired,
			"CONNECT_REQUIRED",
		)
		return "", &message
	}

	if len(request.Args) != 1 || !validUsername(request.Args[0]) {
		message := protocol.ERR(
			protocol.CodeInvalidUsername,
			"INVALID_USERNAME",
		)
		return "", &message
	}

	return request.Args[0], nil
}

func validUsername(username string) bool {
	if len(username) < 1 || len(username) > 24 {
		return false
	}

	for _, r := range username {
		if !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_') {
			return false
		}
	}

	return true
}

func authenticatePlayer(
	session *Session,
	registry *Registry,
	gameWorld *world.World,
) error {
	if err := registry.Add(session); err != nil {
		return err
	}

	roomID, err := gameWorld.GetPlayerReconnectRoom(session.PlayerName)
	if err != nil {
		if errors.Is(err, world.ErrPlayerNotInActiveCombat) {
			roomID = initialRoomID
		} else {
			registry.Remove(session.ID)
			return err
		}
	}

	if err := gameWorld.AddPlayer(session.PlayerName, roomID); err != nil {
		registry.Remove(session.ID)
		return err
	}

	return nil
}

// handleConnect processes a pre-authentication line. It returns false when the
// connection should be closed; recoverable request and name errors keep it open.
func (handler *connectionHandler) handleConnect(line string) bool {
	username, rejection := parseConnectRequest(line)
	if rejection != nil {
		return SendMessage(handler.session, *rejection) == nil
	}

	handler.session.PlayerName = username
	if err := authenticatePlayer(handler.session, handler.registry, handler.gameWorld); err != nil {
		if errors.Is(err, ErrNameInUse) {
			return SendMessage(
				handler.session,
				protocol.ERR(protocol.CodeNameInUse, "NAME_IN_USE"),
			) == nil
		}

		log.Printf("Failed to authenticate player %q: %v", username, err)
		_ = SendMessage(
			handler.session,
			protocol.ERR(protocol.CodeConnectionFailed, "CONNECTION_FAILED"),
		)
		return false
	}

	handler.registered = true
	handler.authenticated = true
	handler.lease.Authenticate()
	handler.stopAuthenticationTimer()

	log.Printf("Player %q connected from %s", username, handler.sessionID)
	roomID, err := handler.gameWorld.GetPlayerRoom(username)
	if err != nil {
		log.Printf("Failed to get room for %q: %v", username, err)
		return false
	}
	players, err := handler.gameWorld.GetPlayersInRoom(roomID)
	if err != nil {
		log.Printf("Failed to get players in room %q: %v", roomID, err)
		return false
	}
	if handler.eventPublisher != nil {
		handler.eventPublisher.Publish(world.PlayerEntered{
			PlayerName:  username,
			RoomID:      roomID,
			RoomPlayers: players,
		})
	}
	handler.enterBroadcast = true

	if err := SendMessage(handler.session, protocol.OK("connected")); err != nil {
		log.Printf("Failed to queue CONNECT response for %s: %v", username, err)
		return false
	}
	handler.connected = true
	return true
}

func (handler *connectionHandler) stopAuthenticationTimer() {
	if !handler.authTimer.Stop() {
		select {
		case <-handler.authTimer.C:
		default:
		}
	}
	handler.authTimeout = nil
}
