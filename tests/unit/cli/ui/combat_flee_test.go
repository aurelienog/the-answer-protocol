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

func TestCombatFleeUpdatesPlayerCombatStateAndViews(t *testing.T) {
	model := ui.NewModel(newCombatFleeTestClient(t))
	model, cmd := enterText(t, model, "Alice")
	if cmd == nil {
		t.Fatal("authentication command is nil")
	}

	model, cmd = updateModel(t, model, cmd())
	if cmd == nil {
		t.Fatal("authentication did not start listening for server events")
	}
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 120, Height: 60})

	event := nextMessage(t, cmd)
	model, _ = updateModel(t, model, event)
	model, _ = updateModel(t, model, ui.TransitionMsg{Phase: ui.TransitionReveal})
	model, _ = updateModel(t, model, ui.TransitionMsg{Phase: ui.TransitionComplete})

	if view := model.View(); !strings.Contains(view, "COMBAT FLEE") {
		t.Fatalf("combat view before fleeing = %q, want combat actions", view)
	}

	model, cmd = enterText(t, model, "COMBAT FLEE")
	if cmd == nil {
		t.Fatal("COMBAT FLEE did not send a command")
	}
	model, _ = updateModel(t, model, cmd())

	view := model.View()
	for _, expected := range []string{
		"Status: ENDED",
		"You: NOT IN COMBAT",
		"You are not in combat",
		"ESCAPED",
	} {
		if !strings.Contains(view, expected) {
			t.Errorf("view after fleeing = %q, want %q", view, expected)
		}
	}
	for _, unavailable := range []string{"COMBAT FLEE", "COMBAT ATTACK"} {
		if strings.Contains(view, unavailable) {
			t.Errorf("view after fleeing still offers %q: %q", unavailable, view)
		}
	}
}

func newCombatFleeTestClient(t *testing.T) *network.Client {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for combat flee test client: %v", err)
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
			var response string
			switch scanner.Text() {
			case "CONNECT Alice":
				response = "OK connected"
				if _, err := fmt.Fprintln(conn, response); err != nil {
					t.Errorf("send authentication response: %v", err)
					return
				}
				if _, err := fmt.Fprintln(
					conn,
					`EVT COMBAT STARTED combat=combat:1 participants=[{"id":"Alice","name":"Alice","is_player":true,"hp":100,"max_hp":100,"alive":true},{"id":"Goblin","name":"Goblin","is_player":false,"hp":50,"max_hp":50,"alive":true}]`,
				); err != nil {
					t.Errorf("send combat started event: %v", err)
					return
				}
				continue
			case "COMBAT FLEE":
				response = "OK fled=true damage=10 hp=90 combat=combat:1"
			default:
				response = "ERR 400 INVALID_ARGUMENTS"
			}

			if _, err := fmt.Fprintln(conn, response); err != nil {
				t.Errorf("send response for %q: %v", scanner.Text(), err)
				return
			}
		}
	}()

	connection, err := network.Dial(context.Background(), listener.Addr().String())
	if err != nil {
		_ = listener.Close()
		t.Fatalf("dial combat flee test server: %v", err)
	}

	client, err := network.NewClient(context.Background(), connection)
	if err != nil {
		_ = connection.Close()
		_ = listener.Close()
		t.Fatalf("create combat flee test client: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
		_ = listener.Close()
	})

	return client
}
