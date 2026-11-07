package internal_test

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/events"
	"the-answer-protocol/internal/server"
	"the-answer-protocol/internal/world"
)

const connectionTestTimeout = 2 * time.Second

// connectionTestEnv contains the resources needed to test a connection.

type connectionTestEnv struct {
	client     net.Conn
	reader     *bufio.Reader
	registry   *server.Registry
	gameWorld  *world.World
	dispatcher *commands.Dispatcher
	eventBus   *events.Bus
}

// newConnectionTestEnv creates a client/server connection pair and starts
// the connection handler.

func newConnectionTestEnv(t *testing.T) *connectionTestEnv {
	t.Helper()

	client, serverConn := net.Pipe()

	registry := server.NewRegistry()

	gameWorld, err := world.Load("../../data/world.json")
	if err != nil {
		client.Close()
		serverConn.Close()
		t.Fatalf("load world: %v", err)
	}

	dispatcher := commands.NewDispatcher()
	commands.RegisterCommands(dispatcher)

	eventBus := events.NewBus()
	notificationDispatcher := server.NewNotificationDispatcher(registry)
	eventBus.Subscribe(notificationDispatcher.Handle)

	go server.HandleConnection(
		serverConn,
		dispatcher,
		registry,
		gameWorld,
		eventBus,
	)

	t.Cleanup(func() {
		_ = client.Close()
		_ = serverConn.Close()
	})

	if err := client.SetDeadline(time.Now().Add(connectionTestTimeout)); err != nil {
		t.Fatalf("set client deadline: %v", err)
	}

	return &connectionTestEnv{
		client:     client,
		reader:     bufio.NewReader(client),
		registry:   registry,
		gameWorld:  gameWorld,
		dispatcher: dispatcher,
		eventBus:   eventBus,
	}
}

// readResponse reads one LF-terminated protocol message.

func (env *connectionTestEnv) readResponse(t *testing.T) string {
	t.Helper()

	line, err := env.reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
}

// sendRequest sends one LF-terminated protocol request.

func (env *connectionTestEnv) sendRequest(t *testing.T, request string) {
	t.Helper()

	if _, err := fmt.Fprintf(env.client, "%s\n", request); err != nil {
		t.Fatalf("send request %q: %v", request, err)
	}
}

func connectTestPlayer(t *testing.T, env *connectionTestEnv, name string) {
	t.Helper()

	if got := env.readResponse(t); got != "OK hello proto=1" {
		t.Fatalf("%s greeting = %q", name, got)
	}
	env.sendRequest(t, "CONNECT "+name)
	if got := env.readResponse(t); got != "OK connected" {
		t.Fatalf("%s connect response = %q", name, got)
	}
}

func (env *connectionTestEnv) sessionOutboxLength(t *testing.T, name string) int {
	t.Helper()

	session, ok := env.registry.GetByName(name)
	if !ok {
		t.Fatalf("session for %q is not registered", name)
	}
	return len(session.Outbox)
}

func assertPlayerRemoved(t *testing.T, env *connectionTestEnv, name string) {
	t.Helper()

	if _, ok := env.registry.GetByName(name); ok {
		t.Errorf("player %q remains in the registry after disconnect", name)
	}

	if roomID, err := env.gameWorld.GetPlayerRoom(name); err == nil {
		t.Errorf("player %q remains in room %q after disconnect", name, roomID)
	}
}

// connectionTestAddr provides a unique remote address for each test connection.

type connectionTestAddr string

func (a connectionTestAddr) Network() string {
	return "test"
}

func (a connectionTestAddr) String() string {
	return string(a)
}

type connectionTestConn struct {
	net.Conn
	remoteAddr net.Addr
}

func (c *connectionTestConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func newSharedConnectionTestEnv(
	t *testing.T,
	registry *server.Registry,
	dispatcher *commands.Dispatcher,
	gameWorld *world.World,
	eventBus *events.Bus,
	id string,
) *connectionTestEnv {
	t.Helper()

	client, serverConn := net.Pipe()

	wrappedConn := &connectionTestConn{
		Conn:       serverConn,
		remoteAddr: connectionTestAddr(id),
	}

	go server.HandleConnection(
		wrappedConn,
		dispatcher,
		registry,
		gameWorld,
		eventBus,
	)

	t.Cleanup(func() {
		_ = client.Close()
		_ = serverConn.Close()
	})

	if err := client.SetDeadline(time.Now().Add(connectionTestTimeout)); err != nil {
		t.Fatalf("set client deadline: %v", err)
	}

	return &connectionTestEnv{
		client:     client,
		reader:     bufio.NewReader(client),
		registry:   registry,
		gameWorld:  gameWorld,
		dispatcher: dispatcher,
		eventBus:   eventBus,
	}
}
