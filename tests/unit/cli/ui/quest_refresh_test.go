package ui_test

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"the-answer-protocol/frontend/cli/network"
	"the-answer-protocol/frontend/cli/ui"
)

func TestQuestCompletionEventRefreshesSidebarQuests(t *testing.T) {
	model := ui.NewModel(newQuestCompletionClient(t))
	model, cmd := enterText(t, model, "Alice")
	if cmd == nil {
		t.Fatal("authentication command is nil")
	}
	model, cmd = updateModel(t, model, cmd())
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 120, Height: 40})

	event := nextMessage(t, cmd)
	model, cmd = updateModel(t, model, event)
	if cmd == nil {
		t.Fatal("quest completion event did not request updated quests")
	}

	model, _ = updateModel(t, model, nextMessage(t, cmd))

	view := model.View()
	if !strings.Contains(view, "Explore Arvendale") {
		t.Fatalf("sidebar = %q, want updated quest name", view)
	}
	if !strings.Contains(view, "completed") {
		t.Fatalf("sidebar = %q, want completed quest status", view)
	}
	if !strings.Contains(view, "WORLD MAP") {
		t.Fatalf("main view = %q, want to remain on the world map", view)
	}
}

func newQuestCompletionClient(t *testing.T) *network.Client {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for quest completion client: %v", err)
	}

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		if _, err := fmt.Fprintln(conn, "OK hello proto=1"); err != nil {
			t.Errorf("send greeting: %v", err)
			return
		}

		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			switch scanner.Text() {
			case "CONNECT Alice":
				if _, err := fmt.Fprintln(conn, "OK connected"); err != nil {
					t.Errorf("send authentication response: %v", err)
					return
				}
				if _, err := fmt.Fprintln(conn, "EVT QUEST COMPLETED quest=quest.explore_arvendale"); err != nil {
					t.Errorf("send quest completion event: %v", err)
					return
				}
			case "QUESTS":
				if _, err := fmt.Fprintln(conn, `OK [{"id":"quest.explore_arvendale","name":"Explore Arvendale","status":"completed"}]`); err != nil {
					t.Errorf("send quest list: %v", err)
					return
				}
			default:
				t.Errorf("unexpected request: %q", scanner.Text())
				return
			}
		}
	}()

	connection, err := network.Dial(context.Background(), listener.Addr().String())
	if err != nil {
		_ = listener.Close()
		t.Fatalf("dial quest completion test server: %v", err)
	}

	client, err := network.NewClient(context.Background(), connection)
	if err != nil {
		_ = connection.Close()
		_ = listener.Close()
		t.Fatalf("create quest completion test client: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
		_ = listener.Close()
	})

	return client
}
