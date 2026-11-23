package internal_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"log/slog"
	"the-answer-protocol/frontend/cli/network"
	"the-answer-protocol/frontend/cli/ui"
	"the-answer-protocol/internal/abuse"
	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/events"
	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/server"
	"the-answer-protocol/internal/world"
)

func TestCLIClientRepeatedMoveNorthSouth(t *testing.T) {
	const moveCount = 10

	gameWorld, err := world.Load("../../data/world.json")
	if err != nil {
		t.Fatalf("load world: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})

	registry := server.NewRegistry(slog.Default())
	dispatcher := commands.NewDispatcher(slog.Default())
	commands.RegisterCommands(dispatcher)
	eventBus := events.NewBus()
	notificationDispatcher := server.NewNotificationDispatcher(registry, slog.Default())
	eventBus.Subscribe(notificationDispatcher.Handle)

	guard := abuse.NewAbuseGuard(abuse.DefaultConfig())
	accepted := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			accepted <- err
			return
		}

		lease, rejection := guard.AcquireConnection(conn.RemoteAddr())
		if rejection != "" {
			_ = conn.Close()
			accepted <- fmt.Errorf("admit test connection: %s", rejection)
			return
		}

		accepted <- nil
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
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connection, err := network.Dial(ctx, listener.Addr().String())
	if err != nil {
		t.Fatalf("CLI Dial: %v", err)
	}
	client, err := network.NewClient(ctx, connection)
	if err != nil {
		t.Fatalf("create CLI client: %v", err)
	}
	defer client.Close()

	select {
	case err := <-accepted:
		if err != nil {
			t.Fatalf("accept CLI connection: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("server did not accept CLI connection: %v", ctx.Err())
	}

	if err := client.Authenticate("Mover"); err != nil {
		t.Fatalf("CLI authenticate: %v", err)
	}

	for i := 0; i < moveCount; i++ {
		direction := "north"
		wantRoom := "market_street"
		if i%2 == 1 {
			direction = "south"
			wantRoom = "harbor_square"
		}

		response, err := client.Send(ctx, "MOVE "+direction)
		if err != nil {
			t.Fatalf("MOVE %s iteration %d: %v", direction, i+1, err)
		}
		if response.Kind != protocol.KindOK || response.Payload != "room="+wantRoom {
			t.Fatalf("MOVE %s iteration %d response = %#v; want OK room=%s",
				direction, i+1, response, wantRoom)
		}

		response, err = client.Send(ctx, "LOOK")
		if err != nil {
			t.Fatalf("LOOK after MOVE %s iteration %d: %v", direction, i+1, err)
		}
		if response.Kind != protocol.KindOK {
			t.Fatalf("LOOK after MOVE %s iteration %d response = %#v; want OK",
				direction, i+1, response)
		}

		var room ui.LookResponse
		if err := json.Unmarshal([]byte(response.Payload), &room); err != nil {
			t.Fatalf("decode LOOK after MOVE %s iteration %d: %v",
				direction, i+1, err)
		}
		if room.Room.ID != wantRoom {
			t.Fatalf("LOOK after MOVE %s iteration %d room = %q; want %q",
				direction, i+1, room.Room.ID, wantRoom)
		}
	}

	select {
	case <-client.Done():
		t.Fatal("CLI client connection closed during repeated movement")
	default:
	}
}

func TestCLIClientConcurrentLookRequests(t *testing.T) {
	const requestCount = 20

	gameWorld, err := world.Load("../../data/world.json")
	if err != nil {
		t.Fatalf("load world: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})

	registry := server.NewRegistry(slog.Default())
	dispatcher := commands.NewDispatcher(slog.Default())
	commands.RegisterCommands(dispatcher)
	eventBus := events.NewBus()
	eventBus.Subscribe(server.NewNotificationDispatcher(registry, slog.Default()).Handle)

	guard := abuse.NewAbuseGuard(abuse.DefaultConfig())
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
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
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connection, err := network.Dial(ctx, listener.Addr().String())
	if err != nil {
		t.Fatalf("CLI Dial: %v", err)
	}
	client, err := network.NewClient(ctx, connection)
	if err != nil {
		t.Fatalf("create CLI client: %v", err)
	}
	defer client.Close()
	if err := client.Authenticate("Concurrent"); err != nil {
		t.Fatalf("CLI authenticate: %v", err)
	}

	start := make(chan struct{})
	results := make(chan error, requestCount)
	var callers sync.WaitGroup
	for i := 0; i < requestCount; i++ {
		callers.Add(1)
		go func() {
			defer callers.Done()
			<-start
			response, err := client.Send(ctx, "LOOK")
			if err == nil {
				if response.Kind != protocol.KindOK {
					err = fmt.Errorf("LOOK response kind = %d, want OK", response.Kind)
				} else {
					var room ui.LookResponse
					if decodeErr := json.Unmarshal([]byte(response.Payload), &room); decodeErr != nil {
						err = fmt.Errorf("decode LOOK response: %w", decodeErr)
					}
				}
			}
			results <- err
		}()
	}
	close(start)
	callers.Wait()
	close(results)

	for err := range results {
		if err != nil {
			t.Error(err)
		}
	}

	select {
	case <-client.Done():
		t.Fatal("CLI client closed during concurrent LOOK requests")
	default:
	}
}
