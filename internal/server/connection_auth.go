package server

import (
	"errors"

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