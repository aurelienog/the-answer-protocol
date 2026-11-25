package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"the-answer-protocol/internal/abuse"
	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/config"
	"the-answer-protocol/internal/events"
	"the-answer-protocol/internal/logger"
	"the-answer-protocol/internal/server"
	"the-answer-protocol/internal/world"
)

func main() {
	log := logger.New()
	cfg := config.LoadConfig(log)
	serverContext, stopServer := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopServer()

	guard := abuse.NewAbuseGuard(abuse.Config{
		MaxConnections:                cfg.MaxConnections,
		MaxConnectionsPerIP:           cfg.MaxConnectionsPerIP,
		MaxConnectionAttemptsPerIP:    cfg.MaxConnectionAttemptsPerIP,
		MaxUnauthenticatedConnections: cfg.MaxUnauthenticatedConnections,
	})
	cleanupContext, stopCleanup := context.WithCancel(context.Background())
	var maintenance sync.WaitGroup
	runMaintenance := func(run func()) {
		maintenance.Add(1)
		go func() {
			defer maintenance.Done()
			run()
		}()
	}
	defer func() {
		stopCleanup()
		maintenance.Wait()
	}()
	runMaintenance(func() { guard.RunCleanup(cleanupContext, time.Minute) })

	gameWorld, err := world.Load(cfg.WorldPath)
	if err != nil {
		log.Error("world.load_failed",
			slog.String("error", err.Error()),
		)
		return
	}
	log.Info("world.loaded",
		slog.Int("rooms", gameWorld.RoomCount()),
	)

	registry := server.NewRegistry(log)
	eventBus := events.NewBus()
	auditLogger := logger.NewAuditLogger(log)
	eventBus.Subscribe(auditLogger.Handle)

	notificationDispatcher := server.NewNotificationDispatcher(registry, log)
	eventBus.Subscribe(notificationDispatcher.Handle)
	combatTurnTimeouts := server.NewCombatTurnTimeoutManager(
		cfg.CombatTurnTimeout,
		commands.NewCombatTurnTimeoutHandler(gameWorld, eventBus, log),
	)
	eventBus.Subscribe(combatTurnTimeouts.Handle)
	runMaintenance(func() { combatTurnTimeouts.Run(cleanupContext) })

	npcRespawns := server.NewNPCRespawnManager(
		server.NewNPCRespawnHandler(gameWorld, eventBus, log),
	)
	eventBus.Subscribe(npcRespawns.Handle)
	runMaintenance(func() { npcRespawns.Run(cleanupContext) })

	dispatcher := commands.NewDispatcher(log)
	commands.RegisterCommands(dispatcher)

	listener, err := net.Listen("tcp", cfg.ServerAddr())
	if err != nil {
		log.Error("server.listen_failed",
			slog.String("address", cfg.ServerAddr()),
			slog.String("error", err.Error()),
		)
		return
	}
	defer listener.Close()

	log.Info("server.listening",
		slog.String("address", cfg.ServerAddr()),
	)

	err = server.Serve(serverContext, listener, func(conn net.Conn) {
		lease, rejection := guard.AcquireConnection(conn.RemoteAddr())
		if rejection != "" {
			log.Warn("connection.rejected",
				slog.String("remote_ip", abuse.RemoteIP(conn.RemoteAddr())),
				slog.String("reason", string(rejection)),
			)
			_ = conn.Close()
			return
		}
		log.Info("connection.accepted",
			slog.String("remote_ip", abuse.RemoteIP(conn.RemoteAddr())),
		)

		server.HandleAdmittedConnection(
			conn,
			dispatcher,
			registry,
			gameWorld,
			eventBus,
			lease,
			cfg.IdleTimeout,
			log,
		)
	})
	if err != nil {
		log.Error("connection.accept_failed",
			slog.String("error", err.Error()),
		)
	}
}
