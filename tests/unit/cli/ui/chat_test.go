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

func TestChatModelInitialView(t *testing.T) {
	model := ui.NewModel(nil)

	view := model.View()
	if !strings.Contains(view, "Welcome, adventurer!") {
		t.Fatalf("initial view = %q, want welcome screen", view)
	}
}

func TestChatModelAcceptsWindowResize(t *testing.T) {
	model := ui.NewModel(nil)

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if updated == nil {
		t.Fatal("Update(WindowSizeMsg) returned a nil model")
	}
	if view := updated.View(); view == "" {
		t.Fatal("resized view is empty")
	}
	if cmd != nil {
		t.Log("resize scheduled a follow-up command")
	}
}

func TestChatCommandOpensHistoryAndTabCyclesFilters(t *testing.T) {
	model := ui.NewModel(newGroupTestClient(t))

	model, cmd := enterText(t, model, "Alice")
	if cmd == nil {
		t.Fatal("authentication command is nil")
	}
	model, _ = updateModel(t, model, cmd())
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 120, Height: 40})

	model, cmd = enterText(t, model, "CHAT")
	if cmd == nil {
		t.Fatal("CHAT did not start the history view transition")
	}
	model, cmd = updateModel(t, model, cmd())
	if cmd == nil {
		t.Fatal("chat view transition did not schedule completion")
	}
	model, _ = updateModel(t, model, cmd())

	view := model.View()
	for _, filter := range []string{"CHAT HISTORY", "[ALL]", "[GLOBAL]", "[ROOM]", "[GROUP]"} {
		if !strings.Contains(view, filter) {
			t.Errorf("chat history view = %q, want it to contain %q", view, filter)
		}
	}

	filterRows := 0
	for _, line := range strings.Split(view, "\n") {
		allPosition := strings.Index(line, "[ALL]")
		if allPosition == -1 {
			continue
		}

		filterRows++
		lastPosition := allPosition
		for _, filter := range []string{"[GLOBAL]", "[ROOM]", "[GROUP]"} {
			position := strings.Index(line, filter)
			if position <= lastPosition {
				t.Errorf("filter row is not in the requested order: %q", line)
				break
			}
			lastPosition = position
		}
	}
	if filterRows < 2 {
		t.Errorf("found %d chat filter rows, want full view and bottom panel", filterRows)
	}

	model, _ = updateModel(t, model, tea.KeyMsg{Type: tea.KeyTab})
	if view := model.View(); !strings.Contains(view, "[GLOBAL]") {
		t.Errorf("view after Tab = %q, want GLOBAL in the filter row", view)
	}
}

func TestChatHistoryRetainsLatestHundredMessagesAndScrolls(t *testing.T) {
	const messageCount = 105

	model := ui.NewModel(newChatHistoryClient(t, messageCount))
	model, cmd := enterText(t, model, "Alice")
	if cmd == nil {
		t.Fatal("authentication command is nil")
	}
	model, cmd = updateModel(t, model, cmd())
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 120, Height: 40})

	for range messageCount {
		event := nextMessage(t, cmd)
		model, cmd = updateModel(t, model, event)
	}

	model, cmd = enterText(t, model, "CHAT")
	if cmd == nil {
		t.Fatal("CHAT did not start the history view transition")
	}
	model, cmd = updateModel(t, model, cmd())
	if cmd == nil {
		t.Fatal("chat view transition did not schedule completion")
	}
	model, _ = updateModel(t, model, cmd())

	view := model.View()
	if !strings.Contains(view, "message-005") {
		t.Errorf("top of chat history = %q, want oldest retained message-005", view)
	}
	if strings.Contains(view, "message-004") {
		t.Errorf("top of chat history contains message-004, which should be outside the 100-message limit")
	}

	model, _ = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEnd})
	if view := model.View(); !strings.Contains(view, "message-104") {
		t.Errorf("bottom of chat history = %q, want newest message-104", view)
	}
}

func newChatHistoryClient(t *testing.T, messageCount int) *network.Client {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for chat history client: %v", err)
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
		if !scanner.Scan() || scanner.Text() != "CONNECT Alice" {
			t.Errorf("first request = %q, want CONNECT Alice", scanner.Text())
			return
		}
		if _, err := fmt.Fprintln(conn, "OK connected"); err != nil {
			t.Errorf("send authentication response: %v", err)
			return
		}

		for i := 0; i < messageCount; i++ {
			scope := "GLOBAL"
			switch i % 3 {
			case 0:
				scope = "GROUP"
			case 1:
				scope = "ROOM"
			}

			if _, err := fmt.Fprintf(
				conn,
				"EVT %s CHAT player=Alice message=message-%03d\n",
				scope,
				i,
			); err != nil {
				t.Errorf("send chat event %d: %v", i, err)
				return
			}
		}

		for scanner.Scan() {
		}
	}()

	connection, err := network.Dial(context.Background(), listener.Addr().String())
	if err != nil {
		_ = listener.Close()
		t.Fatalf("dial chat history server: %v", err)
	}

	client, err := network.NewClient(context.Background(), connection)
	if err != nil {
		_ = connection.Close()
		_ = listener.Close()
		t.Fatalf("create chat history client: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
		_ = listener.Close()
	})

	return client
}

func nextMessage(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()

	for cmd != nil {
		msg := cmd()
		if batch, ok := msg.(tea.BatchMsg); ok {
			if len(batch) == 0 {
				break
			}
			cmd = batch[0]
			continue
		}

		return msg
	}

	t.Fatal("command produced no message")
	return nil
}
