package server

import (
	"errors"
	"log"
	"net"
	"strings"
	"time"

	"the-answer-protocol/internal/abuse"
	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/events"
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

	handler := newConnectionHandler(conn, dispatcher, registry, gameWorld, eventPublisher, lease, idleTimeout)
	handler.run()
}

type connectionHandler struct {
	conn           net.Conn
	dispatcher     *commands.Dispatcher
	registry       *Registry
	gameWorld      *world.World
	eventPublisher events.Publisher
	lease          *abuse.ConnectionLease
	idleTimeout    time.Duration
	session        *Session
	sessionID      string
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

func newConnectionHandler(
	conn net.Conn,
	dispatcher *commands.Dispatcher,
	registry *Registry,
	gameWorld *world.World,
	eventPublisher events.Publisher,
	lease *abuse.ConnectionLease,
	idleTimeout time.Duration,
) *connectionHandler {
	sessionID := conn.RemoteAddr().String()
	return &connectionHandler{
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
		log.Printf("Failed to queue greeting for %s: %v", handler.sessionID, err)
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
			log.Printf("Malformed protocol message from %s: %v", handler.sessionID, event.ProtocolError)
			if err := SendMessage(
				handler.session,
				protocol.ERR(protocol.CodeInvalidRequest, "INVALID_REQUEST"),
			); err != nil {
				log.Printf("Failed to queue malformed-request response for %s: %v", handler.sessionID, err)
				return
			}
			continue
		}
		if event.Err != nil {
			if !errors.Is(event.Err, net.ErrClosed) {
				log.Printf("Read error from %s: %v", handler.sessionID, event.Err)
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
		log.Printf("Authentication timeout from %s", handler.sessionID)
		_ = handler.conn.Close()
		return ReadEvent{}, false
	case <-handler.idleTimeoutChannel():
		log.Printf(
			"Authenticated connection idle timeout for %q from %s",
			handler.session.PlayerName,
			handler.sessionID,
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
	log.Printf("Connection closed: %s", handler.sessionID)
}
