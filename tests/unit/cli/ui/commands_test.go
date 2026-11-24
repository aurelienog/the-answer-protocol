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

func TestQuitCommandExitsAfterServerSuccess(t *testing.T) {
	client, requests := newUICommandClient(t)
	model := ui.NewModel(client)
	model, authCmd := enterText(t, model, "Alice")
	model, _ = updateModel(t, model, authCmd())
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 120, Height: 40})

	model, quitCmd := enterText(t, model, "QUIT")
	if quitCmd == nil {
		t.Fatal("QUIT did not send a command")
	}

	model, resultCmd := updateModel(t, model, quitCmd())
	if resultCmd == nil {
		t.Fatal("successful QUIT did not return a quit command")
	}
	if _, ok := resultCmd().(tea.QuitMsg); !ok {
		t.Fatal("successful QUIT did not terminate the Bubble Tea program")
	}

	if request := <-requests; request != "QUIT" {
		t.Fatalf("server request = %q, want QUIT", request)
	}
}

func TestUseCommandRefreshesInventory(t *testing.T) {
	client, requests := newUICommandClient(t)
	model := ui.NewModel(client)
	model, authCmd := enterText(t, model, "Alice")
	model, _ = updateModel(t, model, authCmd())
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 120, Height: 40})

	model, useCmd := enterText(t, model, "USE Healing Potion")
	if useCmd == nil {
		t.Fatal("USE did not send a command")
	}

	model, inventoryCmd := updateModel(t, model, useCmd())
	if inventoryCmd == nil {
		t.Fatal("successful USE did not request an inventory refresh")
	}

	if request := <-requests; request != "USE Healing Potion" {
		t.Fatalf("server request = %q, want USE Healing Potion", request)
	}

	inventoryResult := inventoryCmd()
	model, _ = updateModel(t, model, inventoryResult)
	if request := <-requests; request != "INVENTORY" {
		t.Fatalf("follow-up request = %q, want INVENTORY", request)
	}
}

func newUICommandClient(t *testing.T) (*network.Client, <-chan string) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for command test client: %v", err)
	}

	requests := make(chan string, 8)
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

			var response string
			switch {
			case request == "CONNECT Alice":
				response = "OK connected"
				if _, err := fmt.Fprintln(conn, response); err != nil {
					t.Errorf("send authentication response: %v", err)
					return
				}
				continue
			case request == "QUIT":
				requests <- request
				response = "OK bye"
			case strings.HasPrefix(request, "USE "):
				requests <- request
				response = "OK used=item.healing_potion changes=hp=10"
			case request == "INVENTORY":
				requests <- request
				response = "OK []"
			default:
				response = "ERR 400 INVALID_ARGUMENTS"
			}

			if _, err := fmt.Fprintln(conn, response); err != nil {
				t.Errorf("send response for %q: %v", request, err)
				return
			}
			if request == "QUIT" {
				return
			}
		}
	}()

	connection, err := network.Dial(context.Background(), listener.Addr().String())
	if err != nil {
		_ = listener.Close()
		t.Fatalf("dial command test server: %v", err)
	}

	client, err := network.NewClient(context.Background(), connection)
	if err != nil {
		_ = connection.Close()
		_ = listener.Close()
		t.Fatalf("create command test client: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
		_ = listener.Close()
	})

	return client, requests
}
