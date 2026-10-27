package server

import (
	"errors"
	"log"
	"net"
	"strings"
	"time"

	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/events"
	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

const (
	initialRoomID         = "harbor_square"
	authenticationTimeout = 30 * time.Second
)

func HandleConnection(
	conn net.Conn,
	dispatcher *commands.Dispatcher,
	registry *Registry,
	gameWorld *world.World,
	eventPublisher events.Publisher,
) {
	sessionID := conn.RemoteAddr().String()

	session := &Session{
		ID:         sessionID,
		Connection: conn,
		PlayerName: "",
		Outbox:     make(chan string, OutboxSize),
	}

	// The handler owns the lifecycle of these channels.
	writerDone := make(chan struct{})
	stopRead := make(chan struct{})

	registered := false
	authenticated := false
	playerEnterBroadcast := false
	quitRequested := false

	go WritePump(session, writerDone)

	readerDone := make(chan struct{})
	readerStarted := false

	defer func() {
		// Stop the reader from sending more events to the handler.
		close(stopRead)

		// Clean up the player's state before broadcasting PLAYER_LEAVE.
		cleanupPlayer(
			sessionID,
			session,
			registry,
			gameWorld,
			registered,
			authenticated,
			playerEnterBroadcast,
			eventPublisher,
		)
		registered = false

		// On QUIT, the final response has already been given a chance to send.
		if quitRequested {
			<-writerDone
		} else {
			// Close the output queue and allow the writer to drain pending messages.
			session.CloseOutbox()
			<-writerDone
		}

		// Closing the connection interrupts a reader blocked in Scan.
		_ = conn.Close()

		// Wait for the reader only if it was started.
		if readerStarted {
			<-readerDone
		}

		log.Printf("Connection closed: %s", sessionID)
	}()

	authTimer := time.NewTimer(authenticationTimeout)
	defer authTimer.Stop()

	// Send the TAP greeting through the single writer.
	if err := SendMessage(session, protocol.OK("hello proto=1")); err != nil {
		log.Printf("Failed to queue greeting for %s: %v", sessionID, err)
		return
	}

	// Start the dedicated TCP reader.
	incoming := make(chan ReadEvent, 16)
	readerStarted = true

	go func() {
		defer close(readerDone)
		ReadPump(conn, incoming, stopRead)
	}()

	connected := false
	authTimeout := authTimer.C

	for {
		var event ReadEvent

		select {
		case <-authTimeout:
			log.Printf("Authentication timeout from %s", sessionID)
			_ = conn.Close()
			return

		case received, ok := <-incoming:
			if !ok {
				return
			}
			event = received
		}

		if event.Err != nil {
			if !errors.Is(event.Err, net.ErrClosed) {
				log.Printf("Read error from %s: %v", sessionID, event.Err)
			}
			return
		}

		line := event.Line
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Only CONNECT is accepted before authentication.
		if !connected {
			username, rejection := parseConnectRequest(line)
			if rejection != nil {
				if SendMessage(session, *rejection) != nil {
					return
				}
				continue
			}

			session.PlayerName = username

			// Register the session first to reserve the unique username.
			if err := authenticatePlayer(session, registry, gameWorld); err != nil {
				if errors.Is(err, ErrNameInUse) {
					if SendMessage(session, protocol.ERR(201, "NAME_IN_USE")) != nil {
						return
					}
					continue
				}

				log.Printf("Failed to authenticate player %q: %v", session.PlayerName, err)

				if SendMessage(session, protocol.ERR(900, "CONNECTION_FAILED")) != nil {
					return
				}
				return
			}

			registered = true

			// The player is fully registered and authenticated.
			authenticated = true

			if err := SendMessage(session, protocol.OK("connected")); err != nil {
				log.Printf("Failed to queue CONNECT response for %s: %v", username, err)
				return
			}

			connected = true

			if !authTimer.Stop() {
				select {
				case <-authTimer.C:
				default:
				}
			}
			authTimeout = nil

			log.Printf("Player %q connected from %s", username, sessionID)
			roomID, err := gameWorld.GetPlayerRoom(username)
			if err != nil {
				log.Printf("Failed to get room for %q: %v", username, err)
				return
			}

			players, err := gameWorld.GetPlayersInRoom(roomID)
			if err != nil {
				log.Printf("Failed to get players in room %q: %v", roomID, err)
				return
			}

			if eventPublisher != nil {
				eventPublisher.Publish(world.PlayerEntered{
					PlayerName:  username,
					RoomID:      roomID,
					RoomPlayers: players,
				})
			}

			playerEnterBroadcast = true
			continue
		}

		// Dispatch commands only after successful authentication.
		ctx := &commands.Context{
			PlayerName: session.PlayerName,
			World:      gameWorld,
			Events:     eventPublisher,
		}

		message, err := dispatcher.Dispatch(ctx, line)

		if errors.Is(err, commands.ErrQuit) {
			quitRequested = true

			// Queue the final response before closing the output queue.
			if queueErr := SendMessage(session, message); queueErr != nil {
				log.Printf(
					"Failed to queue QUIT response for %s: %v",
					session.PlayerName,
					queueErr,
				)
			}

			// Remove the session before Outbox is closed.
			if registered {
				registry.Remove(sessionID)
				registered = false
			}

			session.CloseOutbox()

			// Let writePump send the queued response before returning.
			<-writerDone

			log.Printf("Player %s requested session closure", session.PlayerName)
			return
		}

		if err != nil {
			if errors.Is(err, commands.ErrUnknownCommand) {
				if SendMessage(session, protocol.ERR(408, "UNKNOWN_COMMAND")) != nil {
					return
				}
				continue
			}

			log.Printf("Command error from %s: %v", session.PlayerName, err)

			if SendMessage(session, protocol.ERR(409, "COMMAND_FAILED")) != nil {
				return
			}
			continue
		}

		if err := SendMessage(session, message); err != nil {
			log.Printf(
				"Failed to queue response for %s: %v",
				session.PlayerName,
				err,
			)
			return
		}
	}
}
