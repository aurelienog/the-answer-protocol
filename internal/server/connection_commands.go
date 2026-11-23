package server

import (
	"errors"
	"log/slog"

	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/protocol"
)

// handleCommand rate-limits and dispatches one authenticated request.
// Command and response auditing is centralized in commands.Dispatcher.
// It returns false when the connection should be closed.
func (handler *connectionHandler) handleCommand(line string) bool {
	if !handler.lease.AllowCommand(handler.session.PlayerName, line) {
		handler.log.Warn("abuse.rate_limited",
			slog.String("session_id", handler.sessionID),
			slog.String("ip", handler.remoteIP),
			slog.String("player", handler.session.PlayerName),
			slog.Int("error_code", protocol.CodeRateLimited),
			slog.String("error_name", "RATE_LIMITED"),
		)
		if err := SendMessage(handler.session, protocol.ERR(
			protocol.CodeRateLimited,
			"RATE_LIMITED",
		)); err != nil {
			handler.log.Error("command.send_failed",
				slog.String("session_id", handler.sessionID),
				slog.String("ip", handler.remoteIP),
				slog.String("player", handler.session.PlayerName),
				slog.String("error", err.Error()),
			)
			return false
		}
		return true
	}

	ctx := &commands.Context{
		PlayerName: handler.session.PlayerName,
		World:      handler.gameWorld,
		Events:     handler.eventPublisher,
	}
	message, err := handler.dispatcher.Dispatch(ctx, line)
	if errors.Is(err, commands.ErrQuit) {
		handler.handleQuit(message)
		return false
	}
	if err != nil {
		handler.log.Error("command.dispatch_failed",
			slog.String("session_id", handler.sessionID),
			slog.String("ip", handler.remoteIP),
			slog.String("player", handler.session.PlayerName),
			slog.String("error", err.Error()),
			slog.Int("error_code", protocol.CodeCommandFailed),
			slog.String("error_name", "COMMAND_FAILED"),
		)
		return SendMessage(
			handler.session,
			protocol.ERR(protocol.CodeCommandFailed, "COMMAND_FAILED"),
		) == nil
	}
	if err := SendMessage(handler.session, message); err != nil {
		handler.log.Error("command.send_failed",
			slog.String("session_id", handler.sessionID),
			slog.String("ip", handler.remoteIP),
			slog.String("player", handler.session.PlayerName),
			slog.String("error", err.Error()),
		)
		return false
	}
	return true
}

func (handler *connectionHandler) handleQuit(response protocol.Message) {
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
		handler.log.Error("command.quit_send_failed",
			slog.String("session_id", handler.sessionID),
			slog.String("ip", handler.remoteIP),
			slog.String("player", handler.session.PlayerName),
			slog.String("error", err.Error()),
		)
	}
	handler.session.CloseOutbox()
	<-handler.writerDone
	handler.log.Info("player.quit",
		slog.String("session_id", handler.sessionID),
		slog.String("ip", handler.remoteIP),
		slog.String("player", handler.session.PlayerName),
	)
}