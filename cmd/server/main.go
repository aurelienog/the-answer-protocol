package main

import (
	"context"
	"log"
	"net"
	"time"

	"the-answer-protocol/internal/abuse"
	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/config"
	"the-answer-protocol/internal/events"
	"the-answer-protocol/internal/server"
	"the-answer-protocol/internal/world"
)

func main() {
	cfg := config.LoadConfig()
	guard := abuse.NewAbuseGuard(abuse.Config{
		MaxConnections:                cfg.MaxConnections,
		MaxConnectionsPerIP:           cfg.MaxConnectionsPerIP,
		MaxConnectionAttemptsPerIP:    cfg.MaxConnectionAttemptsPerIP,
		MaxUnauthenticatedConnections: cfg.MaxUnauthenticatedConnections,
	})
	cleanupContext, stopCleanup := context.WithCancel(context.Background())
	defer stopCleanup()
	go guard.RunCleanup(cleanupContext, time.Minute)

	gameWorld, err := world.Load(cfg.WorldPath)
	if err != nil {
		log.Fatalf("Failed to load world: %v", err)
	}
	log.Printf("World loaded: %d rooms", gameWorld.RoomCount())

	registry := server.NewRegistry()
	eventBus := events.NewBus()

	notificationDispatcher := server.NewNotificationDispatcher(registry)
	eventBus.Subscribe(notificationDispatcher.Handle)
	combatTurnTimeouts := server.NewCombatTurnTimeoutManager(
		cfg.CombatTurnTimeout,
		commands.NewCombatTurnTimeoutHandler(gameWorld, eventBus),
	)
	eventBus.Subscribe(combatTurnTimeouts.Handle)
	go combatTurnTimeouts.Run(cleanupContext)

	dispatcher := commands.NewDispatcher()
	commands.RegisterCommands(dispatcher)

	listener, err := net.Listen("tcp", cfg.ServerAddr())
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

		lease, rejection := guard.AcquireConnection(conn.RemoteAddr())
		if rejection != "" {
			log.Printf("Connection rejected from %s: %s", conn.RemoteAddr(), rejection)
			_ = conn.Close()
			continue
		}

		go func(conn net.Conn, lease *abuse.ConnectionLease) {
			server.HandleAdmittedConnection(
				conn,
				dispatcher,
				registry,
				gameWorld,
				eventBus,
				lease,
				cfg.IdleTimeout,
			)
		}(conn, lease)
	}
}
