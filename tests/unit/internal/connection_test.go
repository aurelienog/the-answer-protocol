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

	gameWorld, err := world.Load("../../../data/world.json")
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

	if err := client.SetDeadline(time.Now().Add(testTimeout)); err != nil {
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

func TestHandleConnectionSendsGreeting(t *testing.T) {
	env := newConnectionTestEnv(t)

	got := env.readResponse(t)
	want := "OK hello proto=1"

	if got != want {
		t.Fatalf("greeting = %q, want %q", got, want)
	}
}

func TestHandleConnectionConnectsPlayer(t *testing.T) {
	env := newConnectionTestEnv(t)

	if got := env.readResponse(t); got != "OK hello proto=1" {
		t.Fatalf("greeting = %q", got)
	}

	env.sendRequest(t, "CONNECT Alice")

	if got := env.readResponse(t); got != "OK connected" {
		t.Fatalf("connect response = %q, want %q", got, "OK connected")
	}
}

func TestHandleConnectionRequiresConnectBeforeCommands(t *testing.T) {
	env := newConnectionTestEnv(t)

	_ = env.readResponse(t) // greeting
	env.sendRequest(t, "LOOK")

	got := env.readResponse(t)
	if !strings.HasPrefix(got, "ERR 403 CONNECT_REQUIRED") {
		t.Fatalf("response = %q, want CONNECT_REQUIRED error", got)
	}
}

func TestHandleConnectionRejectsInvalidUsername(t *testing.T) {
	env := newConnectionTestEnv(t)

	_ = env.readResponse(t) // greeting
	env.sendRequest(t, "CONNECT invalid-name")

	got := env.readResponse(t)
	if !strings.HasPrefix(got, "ERR 407 INVALID_USERNAME") {
		t.Fatalf("response = %q, want INVALID_USERNAME error", got)
	}
}

func TestHandleConnectionRejectsUnknownCommand(t *testing.T) {
	env := newConnectionTestEnv(t)

	_ = env.readResponse(t) // greeting
	env.sendRequest(t, "CONNECT Alice")

	if got := env.readResponse(t); got != "OK connected" {
		t.Fatalf("connect response = %q", got)
	}

	env.sendRequest(t, "DANCE")

	got := env.readResponse(t)
	if !strings.HasPrefix(got, "ERR 408 UNKNOWN_COMMAND") {
		t.Fatalf("response = %q, want UNKNOWN_COMMAND error", got)
	}
}

func TestHandleConnectionQuit(t *testing.T) {
	env := newConnectionTestEnv(t)

	_ = env.readResponse(t) // greeting
	env.sendRequest(t, "CONNECT Alice")

	if got := env.readResponse(t); got != "OK connected" {
		t.Fatalf("connect response = %q", got)
	}

	env.sendRequest(t, "QUIT")

	got := env.readResponse(t)
	if got != "OK bye" {
		t.Fatalf("quit response = %q, want %q", got, "OK bye")
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

// connectionTestConn overrides RemoteAddr to give each connection a unique ID.
type connectionTestConn struct {
	net.Conn
	remoteAddr net.Addr
}

func (c *connectionTestConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

// newSharedConnectionTestEnv starts a connection using a shared registry,
// dispatcher and world.
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

	if err := client.SetDeadline(time.Now().Add(testTimeout)); err != nil {
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

func TestHandleConnectionRejectsDuplicateUsername(t *testing.T) {
	first := newConnectionTestEnv(t)

	_ = first.readResponse(t) // greeting
	first.sendRequest(t, "CONNECT Alice")

	if got := first.readResponse(t); got != "OK connected" {
		t.Fatalf("first connect response = %q", got)
	}

	// Use the same registry and world for the second connection.
	second := newSharedConnectionTestEnv(
		t,
		first.registry,
		first.dispatcher,
		first.gameWorld,
		first.eventBus,
		"second-connection",
	)

	_ = second.readResponse(t) // greeting
	second.sendRequest(t, "CONNECT Alice")

	got := second.readResponse(t)
	if !strings.HasPrefix(got, "ERR 201 NAME_IN_USE") {
		t.Fatalf("duplicate username response = %q, want NAME_IN_USE error", got)
	}
}

func TestHandleConnectionBroadcastsPlayerLeave(t *testing.T) {
	observer := newConnectionTestEnv(t)

	_ = observer.readResponse(t) // greeting
	observer.sendRequest(t, "CONNECT Bob")

	if got := observer.readResponse(t); got != "OK connected" {
		t.Fatalf("observer connect response = %q", got)
	}

	player := newSharedConnectionTestEnv(
		t,
		observer.registry,
		observer.dispatcher,
		observer.gameWorld,
		observer.eventBus,
		"player-connection",
	)

	_ = player.readResponse(t) // greeting
	player.sendRequest(t, "CONNECT Alice")

	if got := player.readResponse(t); got != "OK connected" {
		t.Fatalf("player connect response = %q", got)
	}

	// Bob receives the event announcing Alice's arrival.
	enterEvent := observer.readResponse(t)
	if enterEvent != "EVT ROOM_PRESENCE_ENTER Alice" {
		t.Fatalf("enter event = %q, want %q",
			enterEvent, "EVT ROOM_PRESENCE_ENTER Alice")
	}

	player.sendRequest(t, "QUIT")

	if got := player.readResponse(t); got != "OK bye" {
		t.Fatalf("quit response = %q, want %q", got, "OK bye")
	}

	// Bob should be notified when Alice leaves.
	leaveEvent := observer.readResponse(t)
	if leaveEvent != "EVT ROOM_PRESENCE_LEAVE Alice" {
		t.Fatalf("leave event = %q, want %q",
			leaveEvent, "EVT ROOM_PRESENCE_LEAVE Alice")
	}
}

func TestHandleConnectionReleasesUsernameAfterQuit(t *testing.T) {
	first := newConnectionTestEnv(t)

	_ = first.readResponse(t) // greeting
	first.sendRequest(t, "CONNECT Alice")

	if got := first.readResponse(t); got != "OK connected" {
		t.Fatalf("first connect response = %q", got)
	}

	first.sendRequest(t, "QUIT")

	if got := first.readResponse(t); got != "OK bye" {
		t.Fatalf("quit response = %q, want %q", got, "OK bye")
	}

	// Attempt to reuse the same username with the same registry.
	second := newSharedConnectionTestEnv(
		t,
		first.registry,
		first.dispatcher,
		first.gameWorld,
		first.eventBus,
		"reconnected-client",
	)

	_ = second.readResponse(t) // greeting
	second.sendRequest(t, "CONNECT Alice")

	if got := second.readResponse(t); got != "OK connected" {
		t.Fatalf("reconnect response = %q, want %q", got, "OK connected")
	}
}
