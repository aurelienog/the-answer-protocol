package server

import (
	"errors"
	"log/slog"

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

	if len(request.Args) != 1 || !protocol.ValidUsername(request.Args[0]) {
		message := protocol.ERR(
			protocol.CodeInvalidUsername,
			"INVALID_USERNAME",
		)
		return "", &message
	}

	return request.Args[0], nil
}

func authenticatePlayer(
	session *Session,
	registry *Registry,
	gameWorld *world.World,
) (*world.PlayerCountChanged, error) {
	if err := registry.Add(session); err != nil {
		return nil, err
	}

	roomID, err := gameWorld.GetPlayerReconnectRoom(session.PlayerName)
	if err != nil {
		if errors.Is(err, world.ErrPlayerNotInActiveCombat) {
			roomID = initialRoomID
		} else {
			registry.Remove(session.ID)
			return nil, err
		}
	}

	statsEvent, err := gameWorld.AddPlayer(session.PlayerName, roomID)
	if err != nil {
		registry.Remove(session.ID)
		return nil, err
	}

	return statsEvent, nil
}

// handleConnect processes a pre-authentication line. It returns false when the
// connection should be closed; recoverable request and name errors keep it open.
func (handler *connectionHandler) handleConnect(line string) bool {
	username, rejection := parseConnectRequest(line)
	if rejection != nil {
		handler.log.Warn("auth.rejected",
			slog.String("session_id", handler.sessionID),
			slog.String("ip", handler.remoteIP),
			slog.Int("error_code", rejection.Code),
			slog.String("error_name", rejection.Name),
		)
		return SendMessage(handler.session, *rejection) == nil
	}

	handler.session.PlayerName = username

	statsEvent, err := authenticatePlayer(
		handler.session,
		handler.registry,
		handler.gameWorld,
	)
	if err != nil {
		if errors.Is(err, ErrNameInUse) {
			handler.log.Warn("auth.name_in_use",
				slog.String("session_id", handler.sessionID),
				slog.String("ip", handler.remoteIP),
				slog.String("player", username),
				slog.Int("error_code", protocol.CodeNameInUse),
				slog.String("error_name", "NAME_IN_USE"),
			)
			return SendMessage(
				handler.session,
				protocol.ERR(protocol.CodeNameInUse, "NAME_IN_USE"),
			) == nil
		}

		handler.log.Error("auth.failed",
			slog.String("session_id", handler.sessionID),
			slog.String("ip", handler.remoteIP),
			slog.String("player", username),
			slog.String("error", err.Error()),
			slog.Int("error_code", protocol.CodeConnectionFailed),
			slog.String("error_name", "CONNECTION_FAILED"),
		)
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

	roomID, err := handler.gameWorld.GetPlayerRoom(username)
	if err != nil {
		handler.log.Error("auth.room_lookup_failed",
			slog.String("session_id", handler.sessionID),
			slog.String("ip", handler.remoteIP),
			slog.String("player", username),
			slog.String("error", err.Error()),
		)
		return false
	}
	players, err := handler.gameWorld.GetPlayersInRoom(roomID)
	if err != nil {
		handler.log.Error("auth.room_players_failed",
			slog.String("session_id", handler.sessionID),
			slog.String("ip", handler.remoteIP),
			slog.String("room_id", roomID),
			slog.String("error", err.Error()),
		)
		return false
	}

	handler.log.Info("auth.succeeded",
		slog.String("session_id", handler.sessionID),
		slog.String("ip", handler.remoteIP),
		slog.String("player", username),
		slog.String("room_id", roomID),
	)

	if handler.eventPublisher != nil {
		handler.eventPublisher.Publish(world.PlayerEntered{
			PlayerName:  username,
			RoomID:      roomID,
			RoomPlayers: players,
		})
	}
	handler.enterBroadcast = true

	if err := SendMessage(handler.session, protocol.OK("connected")); err != nil {
		handler.log.Error("auth.response_failed",
			slog.String("session_id", handler.sessionID),
			slog.String("ip", handler.remoteIP),
			slog.String("player", username),
			slog.String("error", err.Error()),
		)
		return false
	}
	if statsEvent != nil && handler.eventPublisher != nil {
		handler.eventPublisher.Publish(*statsEvent)
	}
	handler.connected = true
	handler.resetIdleTimer()
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