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

func TestTalkingToNPCRefreshesSidebarQuests(t *testing.T) {
	client, requests := newQuestDiscoveryClient(t)
	model := ui.NewModel(client)
	model, cmd := enterText(t, model, "Alice")
	if cmd == nil {
		t.Fatal("authentication command is nil")
	}
	model, _ = updateModel(t, model, cmd())
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 120, Height: 40})

	model, cmd = enterText(t, model, "TALK Quest Advisor")
	if cmd == nil {
		t.Fatal("TALK did not send a command")
	}
	model, cmd = updateModel(t, model, cmd())
	if request := <-requests; request != "TALK Quest Advisor" {
		t.Fatalf("server request = %q, want TALK Quest Advisor", request)
	}
	model, cmd = updateModel(t, model, cmd())
	model, _ = updateModel(t, model, cmd())

	view := model.View()
	if !strings.Contains(view, "Scout the Realm") {
		t.Fatalf("sidebar = %q, want newly discovered quest", view)
	}
	if !strings.Contains(view, "available") {
		t.Fatalf("sidebar = %q, want available quest status", view)
	}
	if !strings.Contains(view, "CONVERSATION") {
		t.Fatalf("main view = %q, want to show the NPC conversation", view)
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

func newQuestDiscoveryClient(t *testing.T) (*network.Client, <-chan string) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for quest discovery client: %v", err)
	}

	requests := make(chan string, 1)
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
			request := scanner.Text()
			switch request {
			case "CONNECT Alice":
				if _, err := fmt.Fprintln(conn, "OK connected"); err != nil {
					t.Errorf("send authentication response: %v", err)
					return
				}
			case "TALK Quest Advisor":
				requests <- request
				if _, err := fmt.Fprintln(conn, `OK {"npc":{"id":"npc.quest_advisor","name":"Quest Advisor"},"dialogue":["A quest awaits."],"quest":{"id":"quest.scout_the_realm_of_arvendale","name":"Scout the Realm"},"quest_status":"available"}`); err != nil {
					t.Errorf("send NPC conversation: %v", err)
					return
				}
			default:
				t.Errorf("unexpected request: %q", request)
				return
			}
		}
	}()

	connection, err := network.Dial(context.Background(), listener.Addr().String())
	if err != nil {
		_ = listener.Close()
		t.Fatalf("dial quest discovery test server: %v", err)
	}

	client, err := network.NewClient(context.Background(), connection)
	if err != nil {
		_ = connection.Close()
		_ = listener.Close()
		t.Fatalf("create quest discovery test client: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
		_ = listener.Close()
	})

	return client, requests
}
