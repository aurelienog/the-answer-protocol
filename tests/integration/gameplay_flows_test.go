package internal_test

import "testing"

func TestHandleConnectionMovementBroadcastsRoomPresence(t *testing.T) {
	alice := newConnectionTestEnv(t)
	connectTestPlayer(t, alice, "Alice")

	bob := newSharedConnectionTestEnv(
		t, alice.registry, alice.dispatcher, alice.gameWorld, alice.eventBus, "movement-bob",
	)
	connectTestPlayer(t, bob, "Bob")
	if got := alice.readResponse(t); got != "EVT ROOM PRESENCE ENTER Bob" {
		t.Fatalf("Alice's Bob-enter event = %q", got)
	}

	carol := newSharedConnectionTestEnv(
		t, alice.registry, alice.dispatcher, alice.gameWorld, alice.eventBus, "movement-carol",
	)
	connectTestPlayer(t, carol, "Carol")
	for _, observer := range []*connectionTestEnv{alice, bob} {
		if got := observer.readResponse(t); got != "EVT ROOM PRESENCE ENTER Carol" {
			t.Fatalf("room observer's Carol-enter event = %q", got)
		}
	}

	carol.sendRequest(t, "MOVE north")
	if got := carol.readResponse(t); got != "OK room=market_street" {
		t.Fatalf("Carol's move response = %q", got)
	}
	for _, observer := range []*connectionTestEnv{alice, bob} {
		if got := observer.readResponse(t); got != "EVT ROOM PRESENCE LEAVE Carol" {
			t.Fatalf("harbor observer's Carol-leave event = %q", got)
		}
	}

	alice.sendRequest(t, "MOVE north")
	if got := alice.readResponse(t); got != "OK room=market_street" {
		t.Fatalf("Alice's move response = %q", got)
	}
	if got := bob.readResponse(t); got != "EVT ROOM PRESENCE LEAVE Alice" {
		t.Fatalf("Bob's Alice-leave event = %q", got)
	}
	if got := carol.readResponse(t); got != "EVT ROOM PRESENCE ENTER Alice" {
		t.Fatalf("Carol's Alice-enter event = %q", got)
	}

	for player, wantRoom := range map[string]string{
		"Alice": "market_street",
		"Bob":   "harbor_square",
		"Carol": "market_street",
	} {
		if got, err := alice.gameWorld.GetPlayerRoom(player); err != nil || got != wantRoom {
			t.Errorf("GetPlayerRoom(%q) = (%q, %v), want %q", player, got, err, wantRoom)
		}
	}
}

func TestHandleConnectionRoutesRoomAndGlobalChat(t *testing.T) {
	alice := newConnectionTestEnv(t)
	connectTestPlayer(t, alice, "Alice")

	bob := newSharedConnectionTestEnv(
		t, alice.registry, alice.dispatcher, alice.gameWorld, alice.eventBus, "chat-bob",
	)
	connectTestPlayer(t, bob, "Bob")
	if got := alice.readResponse(t); got != "EVT ROOM PRESENCE ENTER Bob" {
		t.Fatalf("Alice's Bob-enter event = %q", got)
	}

	carol := newSharedConnectionTestEnv(
		t, alice.registry, alice.dispatcher, alice.gameWorld, alice.eventBus, "chat-carol",
	)
	connectTestPlayer(t, carol, "Carol")
	for _, observer := range []*connectionTestEnv{alice, bob} {
		if got := observer.readResponse(t); got != "EVT ROOM PRESENCE ENTER Carol" {
			t.Fatalf("room observer's Carol-enter event = %q", got)
		}
	}
	carol.sendRequest(t, "MOVE north")
	if got := carol.readResponse(t); got != "OK room=market_street" {
		t.Fatalf("Carol's move response = %q", got)
	}
	for _, observer := range []*connectionTestEnv{alice, bob} {
		if got := observer.readResponse(t); got != "EVT ROOM PRESENCE LEAVE Carol" {
			t.Fatalf("harbor observer's Carol-leave event = %q", got)
		}
	}

	alice.sendRequest(t, "CHAT ROOM hello room")
	if got := alice.readResponse(t); got != "EVT ROOM CHAT player=Alice message=hello room" {
		t.Fatalf("Alice's room-chat event = %q", got)
	}
	if got := alice.readResponse(t); got != "OK" {
		t.Fatalf("room-chat response = %q", got)
	}
	if got := bob.readResponse(t); got != "EVT ROOM CHAT player=Alice message=hello room" {
		t.Fatalf("Bob's room-chat event = %q", got)
	}
	if got := carol.sessionOutboxLength(t, "Carol"); got != 0 {
		t.Fatalf("Carol received %d room-chat notifications, want 0", got)
	}

	alice.sendRequest(t, "CHAT GLOBAL hello everyone")
	if got := alice.readResponse(t); got != "EVT GLOBAL CHAT player=Alice message=hello everyone" {
		t.Fatalf("Alice's global-chat event = %q", got)
	}
	if got := alice.readResponse(t); got != "OK" {
		t.Fatalf("global-chat response = %q", got)
	}
	for name, recipient := range map[string]*connectionTestEnv{
		"Bob":   bob,
		"Carol": carol,
	} {
		if got := recipient.readResponse(t); got != "EVT GLOBAL CHAT player=Alice message=hello everyone" {
			t.Errorf("%s's global-chat event = %q", name, got)
		}
	}
}

func TestHandleConnectionGroupInviteJoinChatAndLeave(t *testing.T) {
	alice := newConnectionTestEnv(t)
	connectTestPlayer(t, alice, "Alice")

	bob := newSharedConnectionTestEnv(
		t, alice.registry, alice.dispatcher, alice.gameWorld, alice.eventBus, "group-bob",
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

	if group, err := alice.gameWorld.GetPlayerGroup("Bob"); err != nil || len(group.Members) != 2 {
		t.Fatalf("Bob's group = (%+v, %v), want two members", group, err)
	}

	bob.sendRequest(t, "CHAT GROUP hello group")
	if got := bob.readResponse(t); got != "EVT GROUP CHAT player=Bob message=hello group" {
		t.Fatalf("Bob's group-chat event = %q", got)
	}
	if got := bob.readResponse(t); got != "OK" {
		t.Fatalf("group-chat response = %q", got)
	}
	if got := alice.readResponse(t); got != "EVT GROUP CHAT player=Bob message=hello group" {
		t.Fatalf("Alice's group-chat event = %q", got)
	}

	bob.sendRequest(t, "GROUP LEAVE")
	if got := bob.readResponse(t); got != "OK" {
		t.Fatalf("leave-group response = %q", got)
	}
	if got := alice.readResponse(t); got != "EVT GROUP LEAVE player=Bob" {
		t.Fatalf("group-leave event = %q", got)
	}
	if _, err := alice.gameWorld.GetPlayerGroup("Bob"); err == nil {
		t.Error("Bob remains in a group after leaving")
	}
	if group, err := alice.gameWorld.GetPlayerGroup("Alice"); err != nil || len(group.Members) != 1 || group.Members[0] != "Alice" {
		t.Errorf("Alice's group after Bob leaves = (%+v, %v), want Alice alone", group, err)
	}
}
