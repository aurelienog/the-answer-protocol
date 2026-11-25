package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"log/slog"
	"the-answer-protocol/internal/abuse"
	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/events"
	"the-answer-protocol/internal/server"
	"the-answer-protocol/internal/world"
)

const testTimeout = 3 * time.Second
const testInitialRoomID = "harbor_square"

func createTestWorld(t *testing.T) *world.World {
	t.Helper()

	worldData := struct {
		Rooms map[string]world.Room `json:"rooms"`
	}{
		Rooms: map[string]world.Room{
			testInitialRoomID: {
				ID:          testInitialRoomID,
				Name:        "Harbor Square",
				Description: "A small harbor square.",
				Exits:       map[string]string{},
				Items:       map[string]int{},
				NPCs:        []string{},
			},
		},
	}

	data, err := json.Marshal(worldData)
	if err != nil {
		t.Fatalf("marshal test world: %v", err)
	}

	worldFile := filepath.Join(t.TempDir(), "world.json")
	if err := os.WriteFile(worldFile, data, 0600); err != nil {
		t.Fatalf("write test world: %v", err)
	}

	gameWorld, err := world.Load(worldFile)
	if err != nil {
		t.Fatalf("load test world: %v", err)
	}

	return gameWorld
}

func startTestConnection(
	t *testing.T,
) (net.Conn, <-chan struct{}, *server.Registry, *world.World) {
	t.Helper()

	serverConn, clientConn := net.Pipe()

	registry := server.NewRegistry(slog.Default())
	gameWorld := createTestWorld(t)
	eventBus := events.NewBus()
	dispatcher := commands.NewDispatcher(slog.Default())
	commands.RegisterCommands(dispatcher)

	handlerDone := make(chan struct{})
	guard := abuse.NewAbuseGuard(abuse.DefaultConfig())
	lease, rejection := guard.AcquireConnection(serverConn.RemoteAddr())
	if rejection != "" {
		clientConn.Close()
		t.Fatalf("admit test connection: %v", rejection)
	}

	go func() {
		defer close(handlerDone)
		server.HandleAdmittedConnection(
			serverConn,
			dispatcher,
			registry,
			gameWorld,
			eventBus,
			lease,
			0,
			slog.Default(),
		)
	}()

	if err := clientConn.SetDeadline(time.Now().Add(testTimeout)); err != nil {
		clientConn.Close()
		t.Fatalf("set client deadline: %v", err)
	}

	reader := bufio.NewReader(clientConn)

	greeting, err := reader.ReadString('\n')
	if err != nil {
		clientConn.Close()
		t.Fatalf("read greeting: %v", err)
	}

	if greeting != "OK hello proto=1\n" {
		clientConn.Close()
		t.Fatalf("unexpected greeting: %q", greeting)
	}

	t.Cleanup(func() {
		_ = clientConn.Close()

		select {
		case <-handlerDone:
		case <-time.After(testTimeout):
			t.Error("handleConnection did not terminate before timeout")
		}
	})

	return clientConn, handlerDone, registry, gameWorld
}

func readTestMessage(t *testing.T, conn net.Conn, reader *bufio.Reader) string {
	t.Helper()

	if err := conn.SetReadDeadline(time.Now().Add(testTimeout)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	message, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read server message: %v", err)
	}

	return strings.TrimSuffix(strings.TrimSuffix(message, "\n"), "\r")
}

// TestHandleConnectionConnectAndQuit verifies the normal connection lifecycle.
func TestHandleConnectionConnectAndQuit(t *testing.T) {
	clientConn, handlerDone, registry, gameWorld := startTestConnection(t)
	reader := bufio.NewReader(clientConn)

	if _, err := fmt.Fprint(clientConn, "CONNECT Alice\n"); err != nil {
		t.Fatalf("send CONNECT: %v", err)
	}

	if got := readTestMessage(t, clientConn, reader); got != "OK connected" {
		t.Fatalf("unexpected CONNECT response: %q", got)
	}

	if _, exists := registry.GetByName("Alice"); !exists {
		t.Fatal("expected Alice to be present in the session registry")
	}

	roomID, err := gameWorld.GetPlayerRoom("Alice")
	if err != nil {
		t.Fatalf("expected Alice to be present in the world: %v", err)
	}

	if roomID != testInitialRoomID {
		t.Fatalf("expected Alice in room %q, got %q", testInitialRoomID, roomID)
	}

	if _, err := fmt.Fprint(clientConn, "QUIT\n"); err != nil {
		t.Fatalf("send QUIT: %v", err)
	}

	if got := readTestMessage(t, clientConn, reader); got != "OK bye" {
		t.Fatalf("unexpected QUIT response: %q", got)
	}

	select {
	case <-handlerDone:
	case <-time.After(testTimeout):
		t.Fatal("HandleConnection did not terminate after QUIT")
	}

	if _, exists := registry.GetByName("Alice"); exists {
		t.Fatal("expected Alice to be removed from the session registry")
	}

	if _, err := gameWorld.GetPlayerRoom("Alice"); err == nil {
		t.Fatal("expected Alice to be removed from the world")
	}
}

func TestServeShutdownCleansActiveSessions(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	serveFinished := false

	gameWorld := createTestWorld(t)
	registry := server.NewRegistry(slog.Default())
	eventBus := events.NewBus()
	dispatcher := commands.NewDispatcher(slog.Default())
	commands.RegisterCommands(dispatcher)
	guard := abuse.NewAbuseGuard(abuse.DefaultConfig())

	go func() {
		serveDone <- server.Serve(ctx, listener, func(conn net.Conn) {
			lease, rejection := guard.AcquireConnection(conn.RemoteAddr())
			if rejection != "" {
				_ = conn.Close()
				return
			}
			server.HandleAdmittedConnection(
				conn,
				dispatcher,
				registry,
				gameWorld,
				eventBus,
				lease,
				0,
				slog.Default(),
			)
		})
	}()
	t.Cleanup(func() {
		cancel()
		if serveFinished {
			return
		}
		select {
		case <-serveDone:
		case <-time.After(testTimeout):
			t.Error("serve did not stop during test cleanup")
		}
	})

	client, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		cancel()
		t.Fatalf("dial test server: %v", err)
	}
	defer client.Close()
	if err := client.SetDeadline(time.Now().Add(testTimeout)); err != nil {
		t.Fatalf("set client deadline: %v", err)
	}
	reader := bufio.NewReader(client)

	if got := readTestMessage(t, client, reader); got != "OK hello proto=1" {
		t.Fatalf("unexpected greeting: %q", got)
	}
	if _, err := fmt.Fprint(client, "CONNECT Alice\n"); err != nil {
		t.Fatalf("send CONNECT: %v", err)
	}
	if got := readTestMessage(t, client, reader); got != "OK connected" {
		t.Fatalf("unexpected CONNECT response: %q", got)
	}
	if _, exists := registry.GetByName("Alice"); !exists {
		t.Fatal("expected Alice to be registered before shutdown")
	}

	cancel()
	select {
	case serveErr := <-serveDone:
		serveFinished = true
		if serveErr != nil {
			t.Fatalf("serve returned an error during shutdown: %v", serveErr)
		}
	case <-time.After(testTimeout):
		t.Fatal("serve did not stop after cancellation")
	}

	if _, exists := registry.GetByName("Alice"); exists {
		t.Fatal("expected Alice's session to be removed during shutdown")
	}
	if _, err := gameWorld.GetPlayerRoom("Alice"); err == nil {
		t.Fatal("expected Alice to be removed from the world during shutdown")
	}
}

func TestServeReturnsOnUnexpectedAcceptError(t *testing.T) {
	acceptErr := errors.New("listener failure")
	listener := &failingListener{err: acceptErr}

	var handlerCalls atomic.Int32
	err := server.Serve(context.Background(), listener, func(net.Conn) {
		handlerCalls.Add(1)
	})
	if !errors.Is(err, acceptErr) {
		t.Fatalf("serve error = %v, want wrapped %v", err, acceptErr)
	}
	if got := listener.acceptCalls.Load(); got != 1 {
		t.Fatalf("Accept called %d times, want exactly once", got)
	}
	if got := handlerCalls.Load(); got != 0 {
		t.Fatalf("handler called %d times, want zero", got)
	}
}

type failingListener struct {
	err         error
	acceptCalls atomic.Int32
}

func (listener *failingListener) Accept() (net.Conn, error) {
	listener.acceptCalls.Add(1)
	return nil, listener.err
}

func (*failingListener) Close() error {
	return nil
}

func (*failingListener) Addr() net.Addr {
	return &net.TCPAddr{}
}
