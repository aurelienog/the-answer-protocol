package internal_test

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"the-answer-protocol/internal/abuse"
	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/events"
	"the-answer-protocol/internal/server"
	"the-answer-protocol/internal/world"
)

const concurrentClientCount = 100

type concurrentCommandClient struct {
	name       string
	connection net.Conn
	responses  chan string
	readerDone chan struct{}
}

func Test100ConcurrentClientsMutateWorldConcurrently(t *testing.T) {
	gameWorld, err := world.Load("../../data/world.json")
	if err != nil {
		t.Fatalf("load world: %v", err)
	}

	registry := server.NewRegistry()
	dispatcher := commands.NewDispatcher()
	commands.RegisterCommands(dispatcher)
	eventBus := events.NewBus()
	eventBus.Subscribe(server.NewNotificationDispatcher(registry).Handle)

	config := abuse.DefaultConfig()
	config.MaxConnections = concurrentClientCount
	config.MaxConnectionsPerIP = concurrentClientCount
	config.MaxConnectionAttemptsPerIP = concurrentClientCount
	config.MaxUnauthenticatedConnections = concurrentClientCount
	guard := abuse.NewAbuseGuard(config)

	clients := make([]*concurrentCommandClient, concurrentClientCount)
	var readers sync.WaitGroup
	t.Cleanup(func() {
		for _, client := range clients {
			if client != nil {
				_ = client.connection.Close()
			}
		}
		readers.Wait()
	})

	for i := range clients {
		name := fmt.Sprintf("P%03d", i)
		clientConn, serverConn := net.Pipe()
		wrappedConn := &connectionTestConn{
			Conn:       serverConn,
			remoteAddr: connectionTestAddr(fmt.Sprintf("concurrent-client-%03d", i)),
		}
		lease, rejection := guard.AcquireConnection(wrappedConn.RemoteAddr())
		if rejection != "" {
			t.Fatalf("admit client %d: %v", i, rejection)
		}

		client := &concurrentCommandClient{
			name:       name,
			connection: clientConn,
			responses:  make(chan string, concurrentClientCount*3),
			readerDone: make(chan struct{}),
		}
		clients[i] = client
		readers.Add(1)
		go func() {
			defer readers.Done()
			defer close(client.readerDone)
			reader := bufio.NewReader(clientConn)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				client.responses <- strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
			}
		}()

		go server.HandleAdmittedConnection(
			wrappedConn,
			dispatcher,
			registry,
			gameWorld,
			eventBus,
			lease,
			0,
		)
	}

	for _, client := range clients {
		if got := readConcurrentClientResponse(t, client); got != "OK hello proto=1" {
			t.Fatalf("%s greeting = %q", client.name, got)
		}
	}

	var senders sync.WaitGroup
	for _, client := range clients {
		if _, err := fmt.Fprintf(client.connection, "CONNECT %s\n", client.name); err != nil {
			t.Fatalf("send CONNECT for %s: %v", client.name, err)
		}
		if got := readUntilConcurrentClientResponse(t, client, "OK connected"); got != "OK connected" {
			t.Fatalf("%s CONNECT response = %q", client.name, got)
		}
	}

	start := make(chan struct{})
	sendErrors := make(chan error, len(clients))
	for _, client := range clients {
		senders.Add(1)
		go func(client *concurrentCommandClient) {
			defer senders.Done()
			<-start
			_, err := fmt.Fprintf(
				client.connection,
				"GROUP CREATE Group%s\n",
				strings.TrimPrefix(client.name, "P"),
			)
			if err != nil {
				sendErrors <- fmt.Errorf("send GROUP CREATE for %s: %w", client.name, err)
			}
		}(client)
	}
	close(start)
	senders.Wait()
	close(sendErrors)
	for err := range sendErrors {
		t.Error(err)
	}

	for _, client := range clients {
		groupName := "Group" + strings.TrimPrefix(client.name, "P")
		response := readUntilConcurrentClientResponse(t, client, "OK group=")
		if !strings.HasPrefix(response, "OK group=") ||
			!strings.HasSuffix(response, " name="+groupName) {
			t.Fatalf("%s GROUP CREATE response = %q, want created group %q", client.name, response, groupName)
		}

		group, err := gameWorld.GetPlayerGroup(client.name)
		if err != nil {
			t.Fatalf("GetPlayerGroup(%s) after concurrent mutation returned an error: %v", client.name, err)
		}
		if group.Name != groupName || len(group.Members) != 1 || group.Members[0] != client.name {
			t.Fatalf("group for %s = %+v, want %s with only that player", client.name, group, groupName)
		}
	}

	for _, client := range clients {
		if err := client.connection.Close(); err != nil {
			t.Errorf("close client %s: %v", client.name, err)
		}
	}
	for _, client := range clients {
		select {
		case <-client.readerDone:
		case <-time.After(connectionTestTimeout):
			t.Fatalf("reader for %s did not stop", client.name)
		}
	}

	deadline := time.Now().Add(connectionTestTimeout)
	for time.Now().Before(deadline) {
		remaining := false
		for _, client := range clients {
			if _, exists := registry.GetByName(client.name); exists {
				remaining = true
				break
			}
		}
		if !remaining {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("server handlers did not clean up all 100 client sessions")
}

func readConcurrentClientResponse(
	t *testing.T,
	client *concurrentCommandClient,
) string {
	t.Helper()

	select {
	case response := <-client.responses:
		return response
	case <-time.After(connectionTestTimeout):
		t.Fatalf("timed out waiting for %s response", client.name)
		return ""
	}
}

func readUntilConcurrentClientResponse(
	t *testing.T,
	client *concurrentCommandClient,
	prefix string,
) string {
	t.Helper()

	deadline := time.After(connectionTestTimeout)
	for {
		select {
		case response := <-client.responses:
			if strings.HasPrefix(response, prefix) {
				return response
			}
			if strings.HasPrefix(response, "ERR ") {
				t.Fatalf("%s received unexpected error: %q", client.name, response)
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %s response with prefix %q", client.name, prefix)
			return ""
		}
	}
}
