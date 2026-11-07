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

func TestHandleConnectionAbruptDisconnectRemovesPlayerFromGroup(t *testing.T) {
	alice := newConnectionTestEnv(t)
	connectTestPlayer(t, alice, "Alice")

	bob := newSharedConnectionTestEnv(
		t,
		alice.registry,
		alice.dispatcher,
		alice.gameWorld,
		alice.eventBus,
		"group-disconnect-bob",
	)
	connectTestPlayer(t, bob, "Bob")
	if got := alice.readResponse(t); got != "EVT ROOM PRESENCE ENTER Bob" {
		t.Fatalf("Alice's Bob-enter event = %q", got)
	}

	alice.sendRequest(t, "GROUP CREATE Adventurers")
	if got := alice.readResponse(t); got != "OK group=group:1 name=Adventurers" {
		t.Fatalf("create-group response = %q", got)
	}

	alice.sendRequest(t, "GROUP INVITE Bob")
	if got := alice.readResponse(t); got != "OK" {
		t.Fatalf("invite response = %q", got)
	}
	if got := bob.readResponse(t); got != "EVT GROUP INVITE player=Alice group=Adventurers" {
		t.Fatalf("group-invite event = %q", got)
	}

	bob.sendRequest(t, "GROUP JOIN Adventurers")
	if got := bob.readResponse(t); got != "OK group=group:1" {
		t.Fatalf("join-group response = %q", got)
	}
	if got := alice.readResponse(t); got != "EVT GROUP JOIN player=Bob" {
		t.Fatalf("group-join event = %q", got)
	}

	if err := bob.client.Close(); err != nil {
		t.Fatalf("close Bob's connection: %v", err)
	}
	if got := alice.readResponse(t); got != "EVT ROOM PRESENCE LEAVE Bob" {
		t.Fatalf("Alice's Bob-leave event = %q", got)
	}

	group, err := alice.gameWorld.GetPlayerGroup("Alice")
	if err != nil || len(group.Members) != 1 || group.Members[0] != "Alice" {
		t.Fatalf("Alice's group after Bob disconnects = (%+v, %v), want Alice alone", group, err)
	}
	if _, err := alice.gameWorld.GetPlayerGroup("Bob"); err == nil {
		t.Fatal("Bob remains in a group after abruptly disconnecting")
	}

	deadline := time.Now().Add(connectionTestTimeout)
	for time.Now().Before(deadline) {
		if _, ok := alice.registry.GetByName("Bob"); !ok {
			assertPlayerRemoved(t, alice, "Bob")
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("Bob's session remains registered after abrupt disconnect")
}

func TestHandleConnectionDisconnectingGroupCreatorPreservesRemainingMember(t *testing.T) {
	alice := newConnectionTestEnv(t)
	connectTestPlayer(t, alice, "Alice")

	bob := newSharedConnectionTestEnv(
		t,
		alice.registry,
		alice.dispatcher,
		alice.gameWorld,
		alice.eventBus,
		"leader-disconnect-bob",
	)
	connectTestPlayer(t, bob, "Bob")
	if got := alice.readResponse(t); got != "EVT ROOM PRESENCE ENTER Bob" {
		t.Fatalf("Alice's Bob-enter event = %q", got)
	}

	alice.sendRequest(t, "GROUP CREATE Adventurers")
	if got := alice.readResponse(t); got != "OK group=group:1 name=Adventurers" {
		t.Fatalf("create-group response = %q", got)
	}
	alice.sendRequest(t, "GROUP INVITE Bob")
	if got := alice.readResponse(t); got != "OK" {
		t.Fatalf("invite response = %q", got)
	}
	if got := bob.readResponse(t); got != "EVT GROUP INVITE player=Alice group=Adventurers" {
		t.Fatalf("group-invite event = %q", got)
	}
	bob.sendRequest(t, "GROUP JOIN Adventurers")
	if got := bob.readResponse(t); got != "OK group=group:1" {
		t.Fatalf("join-group response = %q", got)
	}
	if got := alice.readResponse(t); got != "EVT GROUP JOIN player=Bob" {
		t.Fatalf("group-join event = %q", got)
	}

	if err := alice.client.Close(); err != nil {
		t.Fatalf("close group creator's connection: %v", err)
	}
	if got := bob.readResponse(t); got != "EVT ROOM PRESENCE LEAVE Alice" {
		t.Fatalf("Bob's Alice-leave event = %q", got)
	}

	group, err := bob.gameWorld.GetPlayerGroup("Bob")
	if err != nil || len(group.Members) != 1 || group.Members[0] != "Bob" {
		t.Fatalf("Bob's group after Alice disconnects = (%+v, %v), want Bob alone", group, err)
	}
	if _, err := bob.gameWorld.GetPlayerGroup("Alice"); err == nil {
		t.Fatal("Alice remains in a group after disconnecting")
	}

	bob.sendRequest(t, "GROUP CREATE New Adventurers")
	if got := bob.readResponse(t); got != "ERR 402 ALREADY_IN_GROUP" {
		t.Fatalf("new group response while Bob remains in the existing group = %q", got)
	}

	deadline := time.Now().Add(connectionTestTimeout)
	for time.Now().Before(deadline) {
		if _, ok := alice.registry.GetByName("Alice"); !ok {
			assertPlayerRemoved(t, alice, "Alice")
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("Alice's session remains registered after disconnecting")
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
