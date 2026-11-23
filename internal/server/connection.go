package server

import (
	"errors"
	"log/slog"
	"net"
	"strings"
	"time"

	"the-answer-protocol/internal/abuse"
	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/events"
	"the-answer-protocol/internal/logger"
	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

const (
	initialRoomID         = world.InitialRoomID
	authenticationTimeout = 30 * time.Second
	readEventBufferSize   = 16
)

// HandleAdmittedConnection handles a connection whose capacity has already
// been reserved by the accept loop.
func HandleAdmittedConnection(
	log *slog.Logger,
	conn net.Conn,
	dispatcher *commands.Dispatcher,
	registry *Registry,
	gameWorld *world.World,
	eventPublisher events.Publisher,
	lease *abuse.ConnectionLease,
	idleTimeout time.Duration,
) {
	if lease == nil {
		_ = conn.Close()
		return
	}

	handler := newConnectionHandler(log, conn, dispatcher, registry, gameWorld, eventPublisher, lease, idleTimeout)
	handler.run()
}

type connectionHandler struct {
	log            *slog.Logger
	conn           net.Conn
	dispatcher     *commands.Dispatcher
	registry       *Registry
	gameWorld      *world.World
	eventPublisher events.Publisher
	lease          *abuse.ConnectionLease
	idleTimeout    time.Duration
	session        *Session
	sessionID      string
	remoteIP       string
	writerDone     chan struct{}
	stopRead       chan struct{}
	readerDone     chan struct{}
	incoming       chan ReadEvent
	readerStarted  bool
	registered     bool
	authenticated  bool
	enterBroadcast bool
	quitRequested  bool
	connected      bool
	authTimer      *time.Timer
	authTimeout    <-chan time.Time
	idleTimer      *time.Timer
}

func remoteIPFromConn(conn net.Conn) string {
	if conn == nil || conn.RemoteAddr() == nil {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		return conn.RemoteAddr().String()
	}
	return host
}

func newConnectionHandler(
	log *slog.Logger,
	conn net.Conn,
	dispatcher *commands.Dispatcher,
	registry *Registry,
	gameWorld *world.World,
	eventPublisher events.Publisher,
	lease *abuse.ConnectionLease,
	idleTimeout time.Duration,
) *connectionHandler {
	if log == nil {
		log = logger.NewNop()
	}

	sessionID := conn.RemoteAddr().String()
	return &connectionHandler{
		log:            log,
		conn:           conn,
		dispatcher:     dispatcher,
		registry:       registry,
		gameWorld:      gameWorld,
		eventPublisher: eventPublisher,
		lease:          lease,
		idleTimeout:    idleTimeout,
		session: &Session{
			ID:         sessionID,
			Connection: conn,
			Outbox:     make(chan string, OutboxSize),
		},
		sessionID:  sessionID,
		remoteIP:   remoteIPFromConn(conn),
		writerDone: make(chan struct{}),
		stopRead:   make(chan struct{}),
		readerDone: make(chan struct{}),
		incoming:   make(chan ReadEvent, readEventBufferSize),
	}
}

func (handler *connectionHandler) run() {
	go WritePump(handler.session, handler.writerDone)
	handler.authTimer = time.NewTimer(authenticationTimeout)
	handler.authTimeout = handler.authTimer.C

	defer handler.lease.Release()
	defer handler.authTimer.Stop()
	defer handler.shutdown()

	if err := SendMessage(handler.session, protocol.OK("hello proto=1")); err != nil {
		handler.log.Error("connection.greeting_failed",
			slog.String("session_id", handler.sessionID),
			slog.String("ip", handler.remoteIP),
			slog.String("error", err.Error()),
		)
		return
	}
	handler.startReader()
	handler.readLoop()
}

func (handler *connectionHandler) startReader() {
	handler.readerStarted = true
	go func() {
		defer close(handler.readerDone)
		ReadPump(handler.conn, handler.incoming, handler.stopRead)
	}()
}

func (handler *connectionHandler) readLoop() {
	for {
		event, ok := handler.nextReadEvent()
		if !ok {
			return
		}
		if event.ProtocolError != nil {
			handler.log.Warn("protocol.malformed",
				slog.String("session_id", handler.sessionID),
				slog.String("ip", handler.remoteIP),
				slog.String("player", handler.session.PlayerName),
				slog.String("error", event.ProtocolError.Error()),
				slog.Int("error_code", protocol.CodeInvalidRequest),
				slog.String("error_name", "INVALID_REQUEST"),
			)
			if err := SendMessage(
				handler.session,
				protocol.ERR(protocol.CodeInvalidRequest, "INVALID_REQUEST"),
			); err != nil {
				handler.log.Error("connection.send_failed",
					slog.String("session_id", handler.sessionID),
					slog.String("ip", handler.remoteIP),
					slog.String("error", err.Error()),
				)
				return
			}
			continue
		}
		if event.Err != nil {
			if !errors.Is(event.Err, net.ErrClosed) {
				handler.log.Warn("connection.read_error",
					slog.String("session_id", handler.sessionID),
					slog.String("ip", handler.remoteIP),
					slog.String("player", handler.session.PlayerName),
					slog.String("error", event.Err.Error()),
				)
			}
			return
		}
		line := event.Line
		if strings.TrimSpace(line) == "" {
			continue
		}
		handler.resetIdleTimer()
		if !handler.connected {
			if !handler.handleConnect(line) {
				return
			}
			continue
		}
		if !handler.handleCommand(line) {
			return
		}
	}
}

func (handler *connectionHandler) nextReadEvent() (ReadEvent, bool) {
	select {
	case <-handler.authTimeout:
		handler.log.Warn("auth.timeout",
			slog.String("session_id", handler.sessionID),
			slog.String("ip", handler.remoteIP),
		)
		_ = handler.conn.Close()
		return ReadEvent{}, false
	case <-handler.idleTimeoutChannel():
		handler.log.Warn("connection.idle_timeout",
			slog.String("session_id", handler.sessionID),
			slog.String("ip", handler.remoteIP),
			slog.String("player", handler.session.PlayerName),
		)
		_ = handler.conn.Close()
		return ReadEvent{}, false
	case event, ok := <-handler.incoming:
		return event, ok
	}
}

func (handler *connectionHandler) idleTimeoutChannel() <-chan time.Time {
	if handler.idleTimer == nil {
		return nil
	}
	return handler.idleTimer.C
}

func (handler *connectionHandler) resetIdleTimer() {
	if !handler.connected || handler.idleTimeout <= 0 {
		return
	}
	if handler.idleTimer == nil {
		handler.idleTimer = time.NewTimer(handler.idleTimeout)
		return
	}
	if !handler.idleTimer.Stop() {
		select {
		case <-handler.idleTimer.C:
		default:
		}
	}
	handler.idleTimer.Reset(handler.idleTimeout)
}

func (handler *connectionHandler) shutdown() {
	if handler.idleTimer != nil {
		if !handler.idleTimer.Stop() {
			select {
			case <-handler.idleTimer.C:
			default:
			}
		}
	}
	close(handler.stopRead)

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

	if !handler.quitRequested {
		handler.session.CloseOutbox()
	}
	<-handler.writerDone
	_ = handler.conn.Close()
	if handler.readerStarted {
		<-handler.readerDone
	}
	handler.log.Info("connection.closed",
		slog.String("session_id", handler.sessionID),
		slog.String("ip", handler.remoteIP),
		slog.String("player", handler.session.PlayerName),
		slog.Bool("quit_requested", handler.quitRequested),
	)
}