package main

import (
	"errors"
	"log"
	"net"
	"strings"

	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/config"
	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/server"
	"the-answer-protocol/internal/world"
)

const initialRoomID = "harbor_square"

func main() {
	cfg := config.LoadConfig()

	gameWorld, err := world.Load("data/world.json")
	if err != nil {
		log.Fatalf("Failed to load world: %v", err)
	}
	log.Printf("World loaded: %d rooms", gameWorld.RoomCount())


	registry := server.NewRegistry()

	dispatcher := commands.NewDispatcher()
	commands.RegisterCommands(dispatcher)

	listener, err := net.Listen("tcp", ":"+cfg.ServerPort)
	if err != nil {
		log.Fatal("Error listening: ", err)
	}
	defer listener.Close()

	log.Printf("TAP server listening on port %s", cfg.ServerPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go handleConnection(conn, dispatcher, registry, gameWorld)
	}
}

func handleConnection(
	conn net.Conn,
	dispatcher *commands.Dispatcher,
	registry *server.Registry,
	gameWorld *world.World,
) {
	sessionID := conn.RemoteAddr().String()

	session := &server.Session{
		ID:          sessionID,
		Connection:  conn,
		PlayerName:  "",
		Outbox:      make(chan string, server.OutboxSize),
	}

	// The handler owns the lifecycle of these channels.
	writerDone := make(chan struct{})
	stopRead := make(chan struct{})

	registered := false
	authenticated := false

	go server.WritePump(session, writerDone)

	defer func() {
		// Stop readPump if it is blocked waiting to deliver an event.
		close(stopRead)

		// Remove the session before broadcasting or closing Outbox.
		if registered {
			registry.Remove(sessionID)
			registered = false
		}

		// Only authenticated players generate a leave event.
		// The session has already been removed, so it won't receive
		// its own departure notification.
		if authenticated {
			// Remove the player from the world before notifying others.
			gameWorld.RemovePlayer(session.PlayerName)

			registry.Broadcast(
				protocol.EVT("PLAYER_LEAVE", session.PlayerName),
			)
		}

		session.CloseOutbox()

		_ = conn.Close()

		// Wait for the writer to finish draining its queue.
		<-writerDone

		log.Printf("Connection closed: %s", sessionID)
	}()

	// Send the TAP greeting through the single writer.
	if err := server.SendMessage(session, protocol.OK("hello proto=1")); err != nil {
		log.Printf("Failed to queue greeting for %s: %v", sessionID, err)
		return
	}

	// Start the dedicated TCP reader.
	incoming := make(chan server.ReadEvent, 16)
	go server.ReadPump(conn, incoming, stopRead)

	connected := false

	for event := range incoming {
		if event.Err != nil {
			if !errors.Is(event.Err, net.ErrClosed) {
				log.Printf("Read error from %s: %v", sessionID, event.Err)
			}
			return
		}

		line := strings.TrimSpace(event.Line)
		if line == "" {
			continue
		}

		// Only CONNECT is accepted before authentication.
		if !connected {
			request, err := protocol.ParseRequest(line)
			if err != nil {
				if server.SendMessage(session, protocol.ERR(400, "INVALID_REQUEST")) != nil {
					return
				}
				continue
			}

			if request.Command != "CONNECT" {
				if server.SendMessage(session, protocol.ERR(400, "CONNECT_REQUIRED")) != nil {
					return
				}
				continue
			}

			if len(request.Args) != 1 {
				if server.SendMessage(session, protocol.ERR(400, "INVALID_USERNAME")) != nil {
					return
				}
				continue
			}

			username := request.Args[0]
			if !validUsername(username) {
				if server.SendMessage(session, protocol.ERR(400, "INVALID_USERNAME")) != nil {
					return
				}
				continue
			}

			session.PlayerName = username

			// Register the session first to reserve the unique username.
			if err := registry.Add(session); err != nil {
				if errors.Is(err, server.ErrNameInUse) {
					if server.SendMessage(session, protocol.ERR(201, "NAME_IN_USE")) != nil {
						return
					}
					continue
				}

				log.Printf("Failed to register player %q: %v", username, err)

				if server.SendMessage(session, protocol.ERR(900, "CONNECTION_FAILED")) != nil {
					return
				}
				return
			}

			registered = true

			// Register the player's location in the world.
			if err := gameWorld.AddPlayer(username, initialRoomID); err != nil {
				log.Printf("Failed to add player %q to world: %v", username, err)

				registry.Remove(sessionID)
				registered = false

				if server.SendMessage(session, protocol.ERR(900, "CONNECTION_FAILED")) != nil {
					return
				}
				return
			}

			// The player is fully registered and authenticated.
			authenticated = true

			if err := server.SendMessage(session, protocol.OK("connected")); err != nil {
				log.Printf("Failed to queue CONNECT response for %s: %v", username, err)
				return
			}

			connected = true

			log.Printf("Player %q connected from %s", username, sessionID)

			registry.Broadcast(protocol.EVT("PLAYER_ENTER", username))
			continue
		}

		// Dispatch commands only after successful authentication.
		ctx := &commands.Context{
			PlayerName: session.PlayerName,
			World:      gameWorld,
		}

		message, err := dispatcher.Dispatch(ctx, line)

		if errors.Is(err, commands.ErrQuit) {
			// Queue the final response before closing the output queue.
			if queueErr := server.SendMessage(session, message); queueErr != nil {
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
				if server.SendMessage(session, protocol.ERR(400, "UNKNOWN_COMMAND")) != nil {
					return
				}
				continue
			}

			log.Printf("Command error from %s: %v", session.PlayerName, err)

			if server.SendMessage(session, protocol.ERR(400, "COMMAND_FAILED")) != nil {
				return
			}
			continue
		}

		if err := server.SendMessage(session, message); err != nil {
			log.Printf(
				"Failed to queue response for %s: %v",
				session.PlayerName,
				err,
			)
			return
		}
	}
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
