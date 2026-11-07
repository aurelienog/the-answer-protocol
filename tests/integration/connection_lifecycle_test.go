package internal_test

import (
	"io"
	"testing"
	"time"
)

func TestHandleConnectionQuit(t *testing.T) {
	env := newConnectionTestEnv(t)

	_ = env.readResponse(t) // greeting
	env.sendRequest(t, "CONNECT Alice")

	if got := env.readResponse(t); got != "OK connected" {
		t.Fatalf("connect response = %q", got)
	}

	env.sendRequest(t, "QUIT")

	got := env.readResponse(t)
	if got != "OK bye" {
		t.Fatalf("quit response = %q, want %q", got, "OK bye")
	}

	assertPlayerRemoved(t, env, "Alice")

	if _, err := env.reader.ReadString('\n'); err != io.EOF {
		t.Fatalf("read after QUIT response error = %v, want EOF", err)
	}
}

func TestHandleConnectionCleansUpAfterAbruptDisconnect(t *testing.T) {
	env := newConnectionTestEnv(t)

	_ = env.readResponse(t) // greeting
	env.sendRequest(t, "CONNECT Alice")
	if got := env.readResponse(t); got != "OK connected" {
		t.Fatalf("connect response = %q, want %q", got, "OK connected")
	}

	if _, ok := env.registry.GetByName("Alice"); !ok {
		t.Fatal("connected player is missing from the registry")
	}
	if roomID, err := env.gameWorld.GetPlayerRoom("Alice"); err != nil || roomID == "" {
		t.Fatalf("GetPlayerRoom(Alice) = (%q, %v), want an active player", roomID, err)
	}

	if err := env.client.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}

	deadline := time.Now().Add(connectionTestTimeout)
	for time.Now().Before(deadline) {
		_, registered := env.registry.GetByName("Alice")
		_, roomErr := env.gameWorld.GetPlayerRoom("Alice")
		if !registered && roomErr != nil {
			return
		}

		time.Sleep(time.Millisecond)
	}

	t.Fatal("player state was not cleaned up after abrupt disconnect")
}

func TestHandleConnectionBroadcastsPlayerLeave(t *testing.T) {
	observer := newConnectionTestEnv(t)

	_ = observer.readResponse(t) // greeting
	observer.sendRequest(t, "CONNECT Bob")

	if got := observer.readResponse(t); got != "OK connected" {
		t.Fatalf("observer connect response = %q", got)
	}

	player := newSharedConnectionTestEnv(
		t,
		observer.registry,
		observer.dispatcher,
		observer.gameWorld,
		observer.eventBus,
		"player-connection",
	)

	_ = player.readResponse(t) // greeting
	player.sendRequest(t, "CONNECT Alice")

	if got := player.readResponse(t); got != "OK connected" {
		t.Fatalf("player connect response = %q", got)
	}

	// Bob receives the event announcing Alice's arrival.
	enterEvent := observer.readResponse(t)
	if enterEvent != "EVT ROOM PRESENCE ENTER Alice" {
		t.Fatalf("enter event = %q, want %q",
			enterEvent, "EVT ROOM PRESENCE ENTER Alice")
	}

	player.sendRequest(t, "QUIT")

	if got := player.readResponse(t); got != "OK bye" {
		t.Fatalf("quit response = %q, want %q", got, "OK bye")
	}

	// Bob should be notified when Alice leaves.
	leaveEvent := observer.readResponse(t)
	if leaveEvent != "EVT ROOM PRESENCE LEAVE Alice" {
		t.Fatalf("leave event = %q, want %q",
			leaveEvent, "EVT ROOM PRESENCE LEAVE Alice")
	}
}

func TestHandleConnectionReleasesUsernameAfterQuit(t *testing.T) {
	first := newConnectionTestEnv(t)

	_ = first.readResponse(t) // greeting
	first.sendRequest(t, "CONNECT Alice")

	if got := first.readResponse(t); got != "OK connected" {
		t.Fatalf("first connect response = %q", got)
	}

	first.sendRequest(t, "QUIT")

	if got := first.readResponse(t); got != "OK bye" {
		t.Fatalf("quit response = %q, want %q", got, "OK bye")
	}

	// Attempt to reuse the same username with the same registry.
	second := newSharedConnectionTestEnv(
		t,
		first.registry,
		first.dispatcher,
		first.gameWorld,
		first.eventBus,
		"reconnected-client",
	)

	_ = second.readResponse(t) // greeting
	second.sendRequest(t, "CONNECT Alice")

	if got := second.readResponse(t); got != "OK connected" {
		t.Fatalf("reconnect response = %q, want %q", got, "OK connected")
	}
}
