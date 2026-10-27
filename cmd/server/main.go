package main

import (
	"log"
	"net"

	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/config"
	"the-answer-protocol/internal/events"
	"the-answer-protocol/internal/server"
	"the-answer-protocol/internal/world"
)

const maxActiveConnections = 3

func main() {
	cfg := config.LoadConfig()

	gameWorld, err := world.Load(cfg.WorldPath)
	if err != nil {
		log.Fatalf("Failed to load world: %v", err)
	}
	log.Printf("World loaded: %d rooms", gameWorld.RoomCount())

	registry := server.NewRegistry()
	eventBus := events.NewBus()

	notificationDispatcher := server.NewNotificationDispatcher(registry)
	eventBus.Subscribe(notificationDispatcher.Handle)

	dispatcher := commands.NewDispatcher()
	commands.RegisterCommands(dispatcher)

	listener, err := net.Listen("tcp", cfg.ServerAddr())
	if err != nil {
		log.Fatal("Error listening: ", err)
	}
	defer listener.Close()

	log.Printf("TAP server listening on port %s", cfg.ServerPort)

	connectionSlots := make(chan struct{}, maxActiveConnections)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		// Reserve a slot before starting a connection handler.
		select {
		case connectionSlots <- struct{}{}:
			// A slot is available.
		default:
			log.Printf(
				"Maximum active connections reached; rejecting %s",
				conn.RemoteAddr(),
			)
			_ = conn.Close()
			continue
		}

		go func(conn net.Conn) {
			defer func() {
				<-connectionSlots
			}()

			server.HandleConnection(conn, dispatcher, registry, gameWorld, eventBus)
		}(conn)
	}
}
