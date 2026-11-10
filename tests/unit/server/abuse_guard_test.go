package server_test

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"strings"
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
		0,
	)
	if _, err := bufio.NewReader(client).ReadString('\n'); !errors.Is(err, io.EOF) {
		t.Fatalf("read from connection without admission lease returned %v, want EOF", err)
	}
}

func TestMalformedFramingReturnsErrorAndKeepsConnectionOpen(t *testing.T) {
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
		t.Fatalf("admit malformed-framing test connection: %v", rejection)
	}
	go server.HandleAdmittedConnection(
		serverConn,
		dispatcher,
		server.NewRegistry(),
		gameWorld,
		nil,
		lease,
		0,
	)
	if err := client.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("set client deadline: %v", err)
	}
	reader := bufio.NewReader(client)
	if greeting, err := reader.ReadString('\n'); err != nil || greeting != "OK hello proto=1\n" {
		t.Fatalf("greeting = (%q, %v)", greeting, err)
	}
	if _, err := client.Write([]byte("CONNECT FramingTester\n")); err != nil {
		t.Fatalf("send CONNECT: %v", err)
	}
	if response, err := reader.ReadString('\n'); err != nil || response != "OK connected\n" {
		t.Fatalf("CONNECT response = (%q, %v)", response, err)
	}

	if _, err := client.Write([]byte{'L', 'O', 'O', 'K', ' ', 0xff, '\n'}); err != nil {
		t.Fatalf("send invalid UTF-8 line: %v", err)
	}
	if response, err := reader.ReadString('\n'); err != nil || response != "ERR 400 INVALID_REQUEST\n" {
		t.Fatalf("invalid UTF-8 response = (%q, %v)", response, err)
	}

	if _, err := client.Write([]byte(strings.Repeat("x", 1025) + "\n")); err != nil {
		t.Fatalf("send oversized line: %v", err)
	}
	if response, err := reader.ReadString('\n'); err != nil || response != "ERR 400 INVALID_REQUEST\n" {
		t.Fatalf("oversized-line response = (%q, %v)", response, err)
	}

	if _, err := client.Write([]byte("LOOK\n")); err != nil {
		t.Fatalf("send valid line after framing errors: %v", err)
	}
	if response, err := reader.ReadString('\n'); err != nil || !strings.HasPrefix(response, "OK ") {
		t.Fatalf("response after framing errors = (%q, %v), want session to remain open", response, err)
	}
}

func TestUnterminatedFrameReturnsProtocolErrorBeforeClosing(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	client, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()
	tcpClient := client.(*net.TCPConn)
	if err := tcpClient.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("set client deadline: %v", err)
	}

	serverConn, err := listener.Accept()
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	gameWorld, err := world.Load("../../../data/world.json")
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	dispatcher := commands.NewDispatcher()
	commands.RegisterCommands(dispatcher)
	guard := abuse.NewAbuseGuard(abuse.Config{})
	lease, rejection := guard.AcquireConnection(serverConn.RemoteAddr())
	if rejection != "" {
		t.Fatalf("admit unterminated-frame test connection: %v", rejection)
	}
	go server.HandleAdmittedConnection(
		serverConn,
		dispatcher,
		server.NewRegistry(),
		gameWorld,
		nil,
		lease,
		0,
	)

	reader := bufio.NewReader(tcpClient)
	if greeting, err := reader.ReadString('\n'); err != nil || greeting != "OK hello proto=1\n" {
		t.Fatalf("greeting = (%q, %v)", greeting, err)
	}
	if _, err := tcpClient.Write([]byte("CONNECT HalfClosed\n")); err != nil {
		t.Fatalf("send CONNECT: %v", err)
	}
	if response, err := reader.ReadString('\n'); err != nil || response != "OK connected\n" {
		t.Fatalf("CONNECT response = (%q, %v)", response, err)
	}
	if _, err := tcpClient.Write([]byte("LOOK")); err != nil {
		t.Fatalf("send unterminated line: %v", err)
	}
	if err := tcpClient.CloseWrite(); err != nil {
		t.Fatalf("half-close client write side: %v", err)
	}
	if response, err := reader.ReadString('\n'); err != nil || response != "ERR 400 INVALID_REQUEST\n" {
		t.Fatalf("unterminated-line response = (%q, %v)", response, err)
	}
	if _, err := reader.ReadString('\n'); !errors.Is(err, io.EOF) {
		t.Fatalf("read after unterminated-line response = %v, want EOF", err)
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
	go server.HandleAdmittedConnection(firstServer, dispatcher, registry, gameWorld, nil, firstLease, 0)
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

func TestAuthenticatedConnectionClosesAfterIdleTimeout(t *testing.T) {
	gameWorld, err := world.Load("../../../data/world.json")
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	dispatcher := commands.NewDispatcher()
	commands.RegisterCommands(dispatcher)
	registry := server.NewRegistry()
	guard := abuse.NewAbuseGuard(abuse.Config{MaxConnections: 1})

	client, serverConn := net.Pipe()
	defer client.Close()
	lease, rejection := guard.AcquireConnection(serverConn.RemoteAddr())
	if rejection != "" {
		t.Fatalf("admit idle-timeout connection: %v", rejection)
	}
	go server.HandleAdmittedConnection(
		serverConn,
		dispatcher,
		registry,
		gameWorld,
		nil,
		lease,
		25*time.Millisecond,
	)
	if err := client.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set client deadline: %v", err)
	}
	reader := bufio.NewReader(client)
	if greeting, err := reader.ReadString('\n'); err != nil || greeting != "OK hello proto=1\n" {
		t.Fatalf("greeting = (%q, %v)", greeting, err)
	}
	if _, err := client.Write([]byte("CONNECT IdlePlayer\n")); err != nil {
		t.Fatalf("send CONNECT: %v", err)
	}
	if response, err := reader.ReadString('\n'); err != nil || response != "OK connected\n" {
		t.Fatalf("CONNECT response = (%q, %v)", response, err)
	}
	if _, err := reader.ReadString('\n'); !errors.Is(err, io.EOF) {
		t.Fatalf("read after idle timeout = %v, want EOF", err)
	}

	deadline := time.Now().Add(time.Second)
	for {
		replacement, rejection := guard.AcquireConnection(serverConn.RemoteAddr())
		if rejection == "" {
			replacement.Release()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("connection capacity was not released after idle timeout: %v", rejection)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestAuthenticatedConnectionActivityResetsIdleTimeout(t *testing.T) {
	gameWorld, err := world.Load("../../../data/world.json")
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	dispatcher := commands.NewDispatcher()
	commands.RegisterCommands(dispatcher)
	client, serverConn := net.Pipe()
	defer client.Close()

	guard := abuse.NewAbuseGuard(abuse.Config{})
	lease, rejection := guard.AcquireConnection(serverConn.RemoteAddr())
	if rejection != "" {
		t.Fatalf("admit idle-reset test connection: %v", rejection)
	}
	go server.HandleAdmittedConnection(
		serverConn,
		dispatcher,
		server.NewRegistry(),
		gameWorld,
		nil,
		lease,
		400*time.Millisecond,
	)
	if err := client.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("set client deadline: %v", err)
	}
	reader := bufio.NewReader(client)
	if greeting, err := reader.ReadString('\n'); err != nil || greeting != "OK hello proto=1\n" {
		t.Fatalf("greeting = (%q, %v)", greeting, err)
	}
	if _, err := client.Write([]byte("CONNECT ActivePlayer\n")); err != nil {
		t.Fatalf("send CONNECT: %v", err)
	}
	if response, err := reader.ReadString('\n'); err != nil || response != "OK connected\n" {
		t.Fatalf("CONNECT response = (%q, %v)", response, err)
	}

	for i := 0; i < 2; i++ {
		time.Sleep(200 * time.Millisecond)
		if _, err := client.Write([]byte("LOOK\n")); err != nil {
			t.Fatalf("send activity %d: %v", i+1, err)
		}
		if response, err := reader.ReadString('\n'); err != nil || len(response) < 3 || response[:3] != "OK " {
			t.Fatalf("LOOK response %d = (%q, %v)", i+1, response, err)
		}
	}

	if _, err := reader.ReadString('\n'); !errors.Is(err, io.EOF) {
		t.Fatalf("connection after final idle period = %v, want EOF", err)
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
		0,
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

func TestAdmittedConnectionRateLimitsGeneralCommands(t *testing.T) {
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
		t.Fatalf("admit command-limit test connection: %v", rejection)
	}
	go server.HandleAdmittedConnection(
		serverConn,
		dispatcher,
		server.NewRegistry(),
		gameWorld,
		nil,
		lease,
		0,
	)
	if err := client.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("set client deadline: %v", err)
	}
	reader := bufio.NewReader(client)
	if greeting, err := reader.ReadString('\n'); err != nil || greeting != "OK hello proto=1\n" {
		t.Fatalf("greeting = (%q, %v)", greeting, err)
	}
	if _, err := client.Write([]byte("CONNECT CommandTester\n")); err != nil {
		t.Fatalf("send CONNECT: %v", err)
	}
	if response, err := reader.ReadString('\n'); err != nil || response != "OK connected\n" {
		t.Fatalf("CONNECT response = (%q, %v)", response, err)
	}

	for i := 0; i < 30; i++ {
		if _, err := client.Write([]byte("LOOK\n")); err != nil {
			t.Fatalf("send command %d: %v", i+1, err)
		}
		if response, err := reader.ReadString('\n'); err != nil || len(response) < 3 || response[:3] != "OK " {
			t.Fatalf("command %d response = (%q, %v), want OK", i+1, response, err)
		}
	}
	if _, err := client.Write([]byte("LOOK\n")); err != nil {
		t.Fatalf("send rate-limited command: %v", err)
	}
	if response, err := reader.ReadString('\n'); err != nil || response != "ERR 429 RATE_LIMITED\n" {
		t.Fatalf("31st command response = (%q, %v), want rate limit", response, err)
	}
}

func TestCombatTurnTimeoutSkipsPlayerTurnAndProcessesNPC(t *testing.T) {
	gameWorld, err := world.Load("../../../data/world.json")
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	if _, err := gameWorld.AddPlayer("Alice", "docks"); err != nil {
		t.Fatalf("AddPlayer(Alice): %v", err)
	}
	combat, _, err := gameWorld.StartCombat("Alice", "Dock Warden")
	if err != nil {
		t.Fatalf("StartCombat(): %v", err)
	}
	combat, turn, err := gameWorld.InitializeCombatTurns(combat.ID)
	if err != nil {
		t.Fatalf("InitializeCombatTurns(): %v", err)
	}
	if turn == nil || !turn.IsPlayer {
		t.Fatalf("initial combat turn = %+v, want Alice's turn", turn)
	}

	manager := server.NewCombatTurnTimeoutManager(
		10*time.Millisecond,
		commands.NewCombatTurnTimeoutHandler(gameWorld, nil),
	)
	defer manager.Stop()
	manager.Handle(*turn)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		combat, err = gameWorld.GetCombat(combat.ID)
		if err != nil {
			t.Fatalf("GetCombat(): %v", err)
		}
		if combat.Round >= 2 {
			if combat.TurnIndex < 0 || !combat.Participants[combat.TurnIndex].IsPlayer {
				t.Fatalf("round 2 turn = %+v, want a player turn after NPC action", combat)
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("combat turn timeout did not advance through the NPC turn")
}

func TestCombatTurnTimeoutIsCancelledWhenTurnChanges(t *testing.T) {
	gameWorld, err := world.Load("../../../data/world.json")
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	if _, err := gameWorld.AddPlayer("Alice", "docks"); err != nil {
		t.Fatalf("AddPlayer(Alice): %v", err)
	}
	combat, _, err := gameWorld.StartCombat("Alice", "Dock Warden")
	if err != nil {
		t.Fatalf("StartCombat(): %v", err)
	}
	combat, turn, err := gameWorld.InitializeCombatTurns(combat.ID)
	if err != nil {
		t.Fatalf("InitializeCombatTurns(): %v", err)
	}
	if turn == nil || !turn.IsPlayer {
		t.Fatalf("initial combat turn = %+v, want Alice's turn", turn)
	}

	manager := server.NewCombatTurnTimeoutManager(
		20*time.Millisecond,
		commands.NewCombatTurnTimeoutHandler(gameWorld, nil),
	)
	defer manager.Stop()
	manager.Handle(*turn)
	manager.Handle(world.CombatTurnChanged{
		CombatID:      combat.ID,
		ParticipantID: "dock_warden",
		IsPlayer:      false,
	})
	time.Sleep(40 * time.Millisecond)

	combat, err = gameWorld.GetCombat(combat.ID)
	if err != nil {
		t.Fatalf("GetCombat(): %v", err)
	}
	if combat.Round != 1 || combat.TurnIndex < 0 ||
		combat.Participants[combat.TurnIndex].ID != "Alice" {
		t.Fatalf("combat changed after obsolete timer was cancelled: %+v", combat)
	}
}

func TestCombatTurnTimeoutIsCancelledWhenCombatEnds(t *testing.T) {
	timeoutExpired := make(chan world.CombatTurnChanged, 1)
	manager := server.NewCombatTurnTimeoutManager(20*time.Millisecond, func(turn world.CombatTurnChanged) {
		timeoutExpired <- turn
	})
	defer manager.Stop()

	turn := world.CombatTurnChanged{
		CombatID:      "combat-1",
		ParticipantID: "Alice",
		IsPlayer:      true,
	}
	manager.Handle(turn)
	manager.Handle(world.CombatEnded{CombatID: turn.CombatID})

	select {
	case expired := <-timeoutExpired:
		t.Fatalf("expired turn callback ran after combat ended: %+v", expired)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestCombatTurnTimeoutManagerStopAndRunCancelTimers(t *testing.T) {
	for _, mode := range []string{"stop", "run"} {
		t.Run(mode, func(t *testing.T) {
			expired := make(chan struct{}, 1)
			manager := server.NewCombatTurnTimeoutManager(
				40*time.Millisecond,
				func(world.CombatTurnChanged) { expired <- struct{}{} },
			)
			manager.Handle(world.CombatTurnChanged{
				CombatID:      "combat-1",
				ParticipantID: "Alice",
				IsPlayer:      true,
			})

			if mode == "stop" {
				manager.Stop()
			} else {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				manager.Run(ctx)
			}

			select {
			case <-expired:
				t.Fatal("timeout callback ran after manager shutdown")
			case <-time.After(80 * time.Millisecond):
			}
		})
	}
}
