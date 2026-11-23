package server

import (
	"errors"
	"log/slog"

	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/protocol"
)

// handleCommand rate-limits and dispatches one authenticated request.
// It returns false when the connection should be closed.
func (handler *connectionHandler) handleCommand(line string) bool {
	playerName := handler.session.GetPlayerName()
	if !handler.lease.AllowCommand(playerName, line) {
		handler.log.Warn("command.rate_limited",
			slog.String("session_id", handler.sessionID),
			slog.String("player", playerName),
		)
		if err := SendMessage(handler.session, protocol.ERR(
			protocol.CodeRateLimited,
			"RATE_LIMITED",
		)); err != nil {
			handler.log.Error("command.response_queue_failed",
				slog.String("session_id", handler.sessionID),
				slog.String("error", err.Error()),
			)
			return false
		}
		return true
	}

	ctx := &commands.Context{
		SessionID:  handler.sessionID,
		PlayerName: playerName,
		World:      handler.gameWorld,
		Events:     handler.eventPublisher,
	}
	message, err := handler.dispatcher.Dispatch(ctx, line)
	if errors.Is(err, commands.ErrQuit) {
		handler.handleQuit(message)
		return false
	}
	if err != nil {
		return SendMessage(
			handler.session,
			protocol.ERR(protocol.CodeCommandFailed, "COMMAND_FAILED"),
		) == nil
	}
	if err := SendMessage(handler.session, message); err != nil {
		handler.log.Error("command.response_queue_failed",
			slog.String("session_id", handler.sessionID),
			slog.String("player", playerName),
			slog.String("error", err.Error()),
		)
		return false
	}
	return true
}

func (handler *connectionHandler) handleQuit(response protocol.Message) {
	playerName := handler.session.GetPlayerName()
	handler.quitRequested = true
	cleanupPlayer(
		handler.log,
		handler.sessionID,
		handler.session,
		handler.registry,
		handler.gameWorld,
		handler.registered,
		handler.authenticated,
		handler.enterBroadcast,
		handler.eventPublisher,
	)
	handler.registered = false
	handler.authenticated = false
	handler.enterBroadcast = false

	if err := SendMessage(handler.session, response); err != nil {
		handler.log.Error("session.quit_response_failed",
			slog.String("session_id", handler.sessionID),
			slog.String("player", playerName),
			slog.String("error", err.Error()),
		)
	}
	handler.session.CloseOutbox()
	<-handler.writerDone
	handler.log.Info("session.quit_requested",
		slog.String("session_id", handler.sessionID),
		slog.String("player", playerName),
	)
}
