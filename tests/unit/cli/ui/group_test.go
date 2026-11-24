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

func TestGroupJoinRemovesAcceptedInvitationAndRefreshesView(t *testing.T) {
	client := newGroupTestClient(t)
	model := ui.NewModel(client)

	model, cmd := enterText(t, model, "Alice")
	if cmd == nil {
		t.Fatal("authentication command is nil")
	}
	model, _ = updateModel(t, model, cmd())
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 120, Height: 40})

	model, cmd = enterText(t, model, "GROUP INVITATIONS")
	if cmd == nil {
		t.Fatal("GROUP INVITATIONS command is nil")
	}
	model, cmd = updateModel(t, model, cmd())
	if cmd == nil {
		t.Fatal("GROUP INVITATIONS did not start the group view transition")
	}
	model, cmd = updateModel(t, model, cmd())
	if cmd == nil {
		t.Fatal("group view transition did not schedule completion")
	}
	model, _ = updateModel(t, model, cmd())

	if view := model.View(); !strings.Contains(view, "[+] Explorers") {
		t.Fatalf("group view = %q, want pending Explorers invitation", view)
	}

	model, cmd = enterText(t, model, "GROUP JOIN Explorers")
	if cmd == nil {
		t.Fatal("GROUP JOIN command is nil")
	}
	model, cmd = updateModel(t, model, cmd())

	if view := model.View(); strings.Contains(view, "[+] Explorers") {
		t.Errorf("group view still shows the accepted invitation: %q", view)
	}
	if view := model.View(); !strings.Contains(view, "[+] Scouts") {
		t.Errorf("group view lost the unrelated pending invitation: %q", view)
	}

	if cmd == nil {
		t.Fatal("GROUP JOIN did not schedule a group information refresh")
	}
	model, _ = updateModel(t, model, cmd())

	if view := model.View(); strings.Contains(view, "[+] Explorers") {
		t.Errorf("refreshed group view still shows the accepted invitation: %q", view)
	}
}

func newGroupTestClient(t *testing.T) *network.Client {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for test client: %v", err)
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
			case "GROUP INVITATIONS":
				response = `OK [{"ID":"group:1","Name":"Explorers"},{"ID":"group:2","Name":"Scouts"}]`
			case "GROUP JOIN Explorers":
				response = "OK group=group:1"
			case "GROUP INFO":
				response = `OK {"ID":"group:1","Name":"Explorers","Members":["Alice"],"Invitations":{}}`
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
		t.Fatalf("dial test server: %v", err)
	}

	client, err := network.NewClient(context.Background(), connection)
	if err != nil {
		_ = connection.Close()
		_ = listener.Close()
		t.Fatalf("create test client: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
		_ = listener.Close()
	})

	return client
}

func enterText(t *testing.T, model ui.Model, text string) (ui.Model, tea.Cmd) {
	t.Helper()

	for _, character := range text {
		model, _ = updateModel(
			t,
			model,
			tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{character}},
		)
	}

	return updateModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})
}

func updateModel(t *testing.T, model ui.Model, msg tea.Msg) (ui.Model, tea.Cmd) {
	t.Helper()

	updated, cmd := model.Update(msg)
	result, ok := updated.(ui.Model)
	if !ok {
		t.Fatalf("Update() returned %T, want ui.Model", updated)
	}

	return result, cmd
}
