package server_test

import (
	"bufio"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"the-answer-protocol/internal/abuse"
	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/server"
	"the-answer-protocol/internal/world"
)

func TestHandleAdmittedConnectionRequiresLease(t *testing.T) {
	client, serverConn := net.Pipe()
	defer client.Close()

	server.HandleAdmittedConnection(
		serverConn,
		commands.NewDispatcher(),
		server.NewRegistry(),
		nil,
		nil,
		nil,
	)
	if _, err := bufio.NewReader(client).ReadString('\n'); !errors.Is(err, io.EOF) {
		t.Fatalf("read from connection without admission lease returned %v, want EOF", err)
	}
}

func TestAdmissionRejectsWhenUnauthenticatedCapacityIsFull(t *testing.T) {
	gameWorld, err := world.Load("../../../data/world.json")
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	dispatcher := commands.NewDispatcher()
	commands.RegisterCommands(dispatcher)
	registry := server.NewRegistry()
	guard := abuse.NewAbuseGuard(abuse.Config{MaxUnauthenticatedConnections: 1})

	firstClient, firstServer := net.Pipe()
	defer firstClient.Close()
	firstLease, rejection := guard.AcquireConnection(firstServer.RemoteAddr())
	if rejection != "" {
		t.Fatalf("admit first connection: %v", rejection)
	}
	go server.HandleAdmittedConnection(firstServer, dispatcher, registry, gameWorld, nil, firstLease)
	if err := firstClient.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set first client deadline: %v", err)
	}
	firstReader := bufio.NewReader(firstClient)
	if greeting, err := firstReader.ReadString('\n'); err != nil || greeting != "OK hello proto=1\n" {
		t.Fatalf("first greeting = (%q, %v)", greeting, err)
	}

	secondClient, secondServer := net.Pipe()
	defer secondClient.Close()
	if _, rejection := guard.AcquireConnection(secondServer.RemoteAddr()); rejection == "" {
		t.Fatal("second connection was admitted despite unauthenticated limit")
	}
	_ = secondServer.Close()
	if _, err := bufio.NewReader(secondClient).ReadString('\n'); !errors.Is(err, io.EOF) {
		t.Fatalf("rejected connection read error = %v, want EOF", err)
	}

	if _, err := firstClient.Write([]byte("CONNECT Alice\n")); err != nil {
		t.Fatalf("send CONNECT: %v", err)
	}
	if response, err := firstReader.ReadString('\n'); err != nil || response != "OK connected\n" {
		t.Fatalf("CONNECT response = (%q, %v)", response, err)
	}
	if _, ok := registry.GetByName("Alice"); !ok {
		t.Fatal("first connection was not authenticated")
	}
}

func TestAdmittedConnectionRateLimitsChatAndKeepsSessionOpen(t *testing.T) {
	gameWorld, err := world.Load("../../../data/world.json")
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	dispatcher := commands.NewDispatcher()
	commands.RegisterCommands(dispatcher)
	client, serverConn := net.Pipe()
	defer client.Close()
	defer serverConn.Close()
	guard := abuse.NewAbuseGuard(abuse.Config{})
	lease, rejection := guard.AcquireConnection(serverConn.RemoteAddr())
	if rejection != "" {
		t.Fatalf("admit chat test connection: %v", rejection)
	}
	go server.HandleAdmittedConnection(
		serverConn,
		dispatcher,
		server.NewRegistry(),
		gameWorld,
		nil,
		lease,
	)
	if err := client.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set client deadline: %v", err)
	}
	reader := bufio.NewReader(client)
	if greeting, err := reader.ReadString('\n'); err != nil || greeting != "OK hello proto=1\n" {
		t.Fatalf("greeting = (%q, %v)", greeting, err)
	}
	if _, err := client.Write([]byte("CONNECT Alice\n")); err != nil {
		t.Fatalf("send CONNECT: %v", err)
	}
	if response, err := reader.ReadString('\n'); err != nil || response != "OK connected\n" {
		t.Fatalf("CONNECT response = (%q, %v)", response, err)
	}

	for i := 0; i < 5; i++ {
		if _, err := client.Write([]byte("CHAT GLOBAL hello\n")); err != nil {
			t.Fatalf("send chat %d: %v", i+1, err)
		}
		if response, err := reader.ReadString('\n'); err != nil || response != "OK\n" {
			t.Fatalf("chat %d response = (%q, %v)", i+1, response, err)
		}
	}
	if _, err := client.Write([]byte("CHAT GLOBAL too-fast\n")); err != nil {
		t.Fatalf("send rate-limited chat: %v", err)
	}
	if response, err := reader.ReadString('\n'); err != nil || response != "ERR 429 RATE_LIMITED\n" {
		t.Fatalf("rate-limited chat response = (%q, %v)", response, err)
	}

	if _, err := client.Write([]byte("LOOK\n")); err != nil {
		t.Fatalf("send command after chat limit: %v", err)
	}
	if response, err := reader.ReadString('\n'); err != nil || len(response) < 3 || response[:3] != "OK " {
		t.Fatalf("response after rate limit = (%q, %v), want session to continue", response, err)
	}
}
