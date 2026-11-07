package server

import (
	"errors"
	"log"

	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/protocol"
)

// handleCommand rate-limits and dispatches one authenticated request.
// It returns false when the connection should be closed.
func (handler *connectionHandler) handleCommand(line string) bool {
	if !handler.lease.AllowCommand(handler.session.PlayerName, line) {
		log.Printf("Rate limit exceeded for player %q", handler.session.PlayerName)
		if err := SendMessage(handler.session, protocol.ERR(
			protocol.CodeRateLimited,
			"RATE_LIMITED",
		)); err != nil {
			log.Printf("Failed to queue rate-limit response for %q: %v", handler.session.PlayerName, err)
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
		log.Printf("Command error from %s: %v", handler.session.PlayerName, err)
		return SendMessage(
			handler.session,
			protocol.ERR(protocol.CodeCommandFailed, "COMMAND_FAILED"),
		) == nil
	}
	if err := SendMessage(handler.session, message); err != nil {
		log.Printf("Failed to queue response for %s: %v", handler.session.PlayerName, err)
		return false
	}
	return true
}

func (handler *connectionHandler) handleQuit(response protocol.Message) {
	handler.quitRequested = true
	cleanupPlayer(
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
		log.Printf("Failed to queue QUIT response for %s: %v", handler.session.PlayerName, err)
	}
	handler.session.CloseOutbox()
	<-handler.writerDone
	log.Printf("Player %s requested session closure", handler.session.PlayerName)
}
