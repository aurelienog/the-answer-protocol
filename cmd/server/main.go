package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
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
		log.Error("world.load_failed",
			slog.String("world_path", cfg.WorldPath),
			slog.String("error", err.Error()),
		)
		return
	}
	log.Info("world.loaded",
		slog.String("world_path", cfg.WorldPath),
		slog.Int("rooms", gameWorld.RoomCount()),
	)

	registry := server.NewRegistry()
	eventBus := events.NewBus()

	auditLogger := server.NewAuditLogger(log)
	eventBus.Subscribe(auditLogger.Handle)

	notificationDispatcher := server.NewNotificationDispatcher(registry)
	eventBus.Subscribe(notificationDispatcher.Handle)
	combatTurnTimeouts := server.NewCombatTurnTimeoutManager(
		cfg.CombatTurnTimeout,
		commands.NewCombatTurnTimeoutHandler(gameWorld, eventBus),
	)
	eventBus.Subscribe(combatTurnTimeouts.Handle)
	go combatTurnTimeouts.Run(cleanupContext)

	dispatcher := commands.NewDispatcher()
	dispatcher.SetLogger(log)
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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Info("server.shutdown_requested", slog.String("signal", sig.String()))
		stopCleanup()
		_ = listener.Close()
	}()

	log.Info("server.listening",
		slog.String("address", cfg.ServerAddr()),
	)

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				log.Info("server.stopped")
				return
			}
			log.Error("connection.accept_failed",
				slog.String("error", err.Error()),
			)
			continue
		}

		ip := conn.RemoteAddr().String()
		if host, _, splitErr := net.SplitHostPort(ip); splitErr == nil {
			ip = host
		}

		lease, rejection := guard.AcquireConnection(conn.RemoteAddr())
		if rejection != "" {
			log.Warn("abuse.connection_rejected",
				slog.String("ip", ip),
				slog.String("remote_addr", conn.RemoteAddr().String()),
				slog.String("reason", string(rejection)),
			)
			_ = conn.Close()
			continue
		}
		log.Info("connection.accepted",
			slog.String("ip", ip),
			slog.String("remote_addr", conn.RemoteAddr().String()),
		)

		go func(conn net.Conn, lease *abuse.ConnectionLease) {
			server.HandleAdmittedConnection(
				log,
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