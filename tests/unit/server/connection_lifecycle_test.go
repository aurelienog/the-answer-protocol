package server_test

import (
	"bufio"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"the-answer-protocol/internal/abuse"
	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/server"
	"the-answer-protocol/internal/world"
)

var authenticationTimeoutTestOnce sync.Once

func TestHandleConnectionClosesUnauthenticatedSessionOnTimeout(t *testing.T) {
	runTimeoutCheck := false
	authenticationTimeoutTestOnce.Do(func() {
		runTimeoutCheck = true
	})
	if !runTimeoutCheck {
		t.Skip("the production authentication timeout is 30 seconds; verify once per test process")
	}

	client, serverConn := net.Pipe()
	defer client.Close()
	defer serverConn.Close()

	gameWorld, err := world.Load("../../../data/world.json")
	if err != nil {
		t.Fatalf("load world: %v", err)
	}

	dispatcher := commands.NewDispatcher()
	commands.RegisterCommands(dispatcher)
	registry := server.NewRegistry()

	guard := abuse.NewAbuseGuard(abuse.DefaultConfig())
	lease, rejection := guard.AcquireConnection(serverConn.RemoteAddr())
	if rejection != "" {
		t.Fatalf("admit test connection: %v", rejection)
	}
	go server.HandleAdmittedConnection(
		serverConn,
		dispatcher,
		registry,
		gameWorld,
		nil,
		lease,
		0,
	)

	if err := client.SetDeadline(time.Now().Add(35 * time.Second)); err != nil {
		t.Fatalf("set client deadline: %v", err)
	}

	reader := bufio.NewReader(client)
	greeting, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read greeting: %v", err)
	}
	if greeting != "OK hello proto=1\n" {
		t.Fatalf("greeting = %q, want %q", greeting, "OK hello proto=1\n")
	}

	if _, err := reader.ReadString('\n'); err != io.EOF {
		t.Fatalf("read after authentication timeout error = %v, want EOF", err)
	}

	if _, ok := registry.GetByName("Alice"); ok {
		t.Error("unauthenticated session registered Alice")
	}
	if _, err := gameWorld.GetPlayerRoom("Alice"); err == nil {
		t.Error("unauthenticated session added Alice to the world")
	}
}
