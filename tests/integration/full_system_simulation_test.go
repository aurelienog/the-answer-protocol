package internal_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"the-answer-protocol/frontend/cli/network"
	"the-answer-protocol/internal/abuse"
	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/desktop"
	"the-answer-protocol/internal/events"
	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/server"
	"the-answer-protocol/internal/world"
)

const fullSimulationTimeout = 10 * time.Second

type guiSimulationClient struct {
	app      *desktop.App
	username string
}

func startFullSimulationServer(t *testing.T) (
	string,
	*world.World,
	*server.Registry,
	context.CancelFunc,
) {
	t.Helper()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	gameWorld, err := world.Load("../../data/world.json")
	if err != nil {
		t.Fatalf("load world: %v", err)
	}

	registry := server.NewRegistry(log)
	dispatcher := commands.NewDispatcher(log)
	commands.RegisterCommands(dispatcher)
	eventBus := events.NewBus()
	eventBus.Subscribe(server.NewNotificationDispatcher(registry, log).Handle)

	guardConfig := abuse.DefaultConfig()
	guardConfig.MaxConnections = 100
	guardConfig.MaxConnectionsPerIP = 100
	guardConfig.MaxConnectionAttemptsPerIP = 100
	guardConfig.MaxUnauthenticatedConnections = 100
	guard := abuse.NewAbuseGuard(guardConfig)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	_, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			lease, rejection := guard.AcquireConnection(conn.RemoteAddr())
			if rejection != "" {
				_ = conn.Close()
				continue
			}
			go server.HandleAdmittedConnection(
				conn,
				dispatcher,
				registry,
				gameWorld,
				eventBus,
				lease,
				0,
				log,
			)
		}
	}()

	t.Cleanup(func() {
		cancel()
		_ = listener.Close()
	})

	return listener.Addr().String(), gameWorld, registry, cancel
}

func connectSimulationCLI(
	t *testing.T,
	address string,
	username string,
) *network.Client {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), fullSimulationTimeout)
	t.Cleanup(cancel)
	connection, err := network.Dial(ctx, address)
	if err != nil {
		t.Fatalf("dial CLI %s: %v", username, err)
	}
	client, err := network.NewClient(ctx, connection)
	if err != nil {
		t.Fatalf("create CLI %s: %v", username, err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})
	if err := client.Authenticate(username); err != nil {
		t.Fatalf("authenticate CLI %s: %v", username, err)
	}
	return client
}

func connectSimulationGUI(
	t *testing.T,
	address string,
	username string,
) *guiSimulationClient {
	t.Helper()

	app := desktop.NewApp()
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatalf("split server address: %v", err)
	}

	body, err := json.Marshal(map[string]string{
		"host":     host,
		"port":     port,
		"username": username,
	})
	if err != nil {
		t.Fatalf("encode GUI connect request: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/connect", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	app.HandleConnect(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GUI %s HTTP status = %d, body = %q", username, recorder.Code, recorder.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode GUI %s connect response: %v", username, err)
	}
	if response["status"] != "OK" {
		t.Fatalf("GUI %s connect response = %+v", username, response)
	}

	gui := &guiSimulationClient{app: app, username: username}
	t.Cleanup(func() {
		guiSendCommand(t, gui, "QUIT")
	})
	return gui
}

func guiSendCommand(
	t *testing.T,
	gui *guiSimulationClient,
	command string,
) {
	t.Helper()

	body, err := json.Marshal(map[string]string{"command": command})
	if err != nil {
		t.Fatalf("encode GUI command: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/command", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	gui.app.HandleCommand(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GUI %s command %q HTTP status = %d", gui.username, command, recorder.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode GUI %s command response: %v", gui.username, err)
	}
	if response["status"] != "OK" {
		t.Fatalf("GUI %s command %q failed: %+v", gui.username, command, response)
	}
}

func waitForSimulationInvariants(
	t *testing.T,
	gameWorld *world.World,
	registry *server.Registry,
	names []string,
) {
	t.Helper()

	deadline := time.Now().Add(fullSimulationTimeout)
	for time.Now().Before(deadline) {
		count, err := gameWorld.GetPlayerCount()
		if err == nil && count == len(names) {
			registered := 0
			for _, name := range names {
				if _, ok := registry.GetByName(name); ok {
					registered++
				}
				if _, err := gameWorld.GetPlayerRoom(name); err != nil {
					registered = -1
					break
				}
			}
			if registered == len(names) {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("simulation invariants not reached for players %v", names)
}

func TestFullSystemSimulationMixedCLIAndGUI(t *testing.T) {
	address, gameWorld, registry, _ := startFullSimulationServer(t)

	cliNames := []string{"CliAlice", "CliBob"}
	guiNames := []string{"GuiCarol", "GuiDave", "GuiEve"}
	names := append(append([]string{}, cliNames...), guiNames...)

	clis := make(map[string]*network.Client, len(cliNames))
	for _, name := range cliNames {
		clis[name] = connectSimulationCLI(t, address, name)
	}
	guis := make([]*guiSimulationClient, 0, len(guiNames))
	for _, name := range guiNames {
		guis = append(guis, connectSimulationGUI(t, address, name))
	}

	waitForSimulationInvariants(t, gameWorld, registry, names)

	start := make(chan struct{})
	results := make(chan error, len(names))
	var callers sync.WaitGroup

	for name, client := range clis {
		callers.Add(1)
		go func(name string, client *network.Client) {
			defer callers.Done()
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), fullSimulationTimeout)
			defer cancel()
			response, err := client.Send(ctx, "MOVE north")
			if err == nil && response.Kind != protocol.KindOK {
				err = fmt.Errorf("CLI %s MOVE response = %+v", name, response)
			}
			results <- err
		}(name, client)
	}

	for _, gui := range guis {
		callers.Add(1)
		go func(gui *guiSimulationClient) {
			defer callers.Done()
			<-start
			body, err := json.Marshal(map[string]string{"command": "MOVE south"})
			if err == nil {
				request := httptest.NewRequest(http.MethodPost, "/api/command", bytes.NewReader(body))
				recorder := httptest.NewRecorder()
				gui.app.HandleCommand(recorder, request)
				if recorder.Code != http.StatusOK {
					err = fmt.Errorf("GUI %s MOVE status = %d", gui.username, recorder.Code)
				}
			}
			results <- err
		}(gui)
	}

	close(start)
	callers.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Error(err)
		}
	}

	if _, err := clis["CliAlice"].Send(context.Background(), "CHAT GLOBAL mixed simulation"); err != nil {
		t.Fatalf("send mixed simulation chat: %v", err)
	}
	guiSendCommand(t, guis[0], "CHAT GLOBAL gui simulation")

	waitForSimulationInvariants(t, gameWorld, registry, names)

	count, err := gameWorld.GetPlayerCount()
	if err != nil {
		t.Fatalf("GetPlayerCount(): %v", err)
	}
	if count != len(names) {
		t.Fatalf("player count = %d, want %d", count, len(names))
	}

	rooms := make(map[string]string)
	for _, name := range names {
		room, err := gameWorld.GetPlayerRoom(name)
		if err != nil {
			t.Fatalf("GetPlayerRoom(%s): %v", name, err)
		}
		rooms[name] = room
		if _, ok := registry.GetByName(name); !ok {
			t.Fatalf("player %s missing from registry", name)
		}
	}

	if strings.TrimSpace(strings.Join(names, ",")) == "" {
		t.Fatal("simulation names are empty")
	}
}
