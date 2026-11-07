package internal_test

import (
	"testing"
)

func TestHandleConnectionGroupChatExcludesNonMembersInSameRoom(t *testing.T) {
	alice := newConnectionTestEnv(t)
	connectTestPlayer(t, alice, "Alice")

	bob := newSharedConnectionTestEnv(
		t, alice.registry, alice.dispatcher, alice.gameWorld, alice.eventBus, "chat-isolation-bob",
	)
	connectTestPlayer(t, bob, "Bob")
	if got := alice.readResponse(t); got != "EVT ROOM PRESENCE ENTER Bob" {
		t.Fatalf("Alice's Bob-enter event = %q", got)
	}

	carol := newSharedConnectionTestEnv(
		t, alice.registry, alice.dispatcher, alice.gameWorld, alice.eventBus, "chat-isolation-carol",
	)
	connectTestPlayer(t, carol, "Carol")
	for _, observer := range []*connectionTestEnv{alice, bob} {
		if got := observer.readResponse(t); got != "EVT ROOM PRESENCE ENTER Carol" {
			t.Fatalf("room observer's Carol-enter event = %q", got)
		}
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

	bob.sendRequest(t, "CHAT GROUP message for our group")
	if got := bob.readResponse(t); got != "EVT GROUP CHAT player=Bob message=message for our group" {
		t.Fatalf("Bob's group-chat event = %q", got)
	}
	if got := bob.readResponse(t); got != "OK" {
		t.Fatalf("Bob's group-chat response = %q", got)
	}
	if got := alice.readResponse(t); got != "EVT GROUP CHAT player=Bob message=message for our group" {
		t.Fatalf("Alice's group-chat event = %q", got)
	}
	if got := carol.sessionOutboxLength(t, "Carol"); got != 0 {
		t.Fatalf("non-member Carol received %d group-chat notifications, want 0", got)
	}

	carol.sendRequest(t, "WHO")
	if got := carol.readResponse(t); got != "OK players=3" {
		t.Fatalf("Carol's response after excluded group chat = %q", got)
	}
}

func TestHandleConnectionChatRejectsInvalidInputAndRemainsUsable(t *testing.T) {
	alice := newConnectionTestEnv(t)
	connectTestPlayer(t, alice, "Alice")

	for _, test := range []struct {
		request string
		want    string
	}{
		{request: "CHAT", want: "ERR 422 INVALID_ARGUMENTS"},
		{request: "CHAT ROOM", want: "ERR 422 INVALID_ARGUMENTS"},
		{request: "CHAT ROOM    ", want: "ERR 422 INVALID_ARGUMENTS"},
		{request: "CHAT UNKNOWN hello", want: "ERR 422 INVALID_ARGUMENTS"},
		{request: "CHAT GROUP hello", want: "ERR 401 NOT_IN_GROUP"},
	} {
		alice.sendRequest(t, test.request)
		if got := alice.readResponse(t); got != test.want {
			t.Errorf("%q response = %q, want %q", test.request, got, test.want)
		}
	}

	alice.sendRequest(t, "WHO")
	if got := alice.readResponse(t); got != "OK players=1" {
		t.Fatalf("response after invalid chat input = %q, want %q", got, "OK players=1")
	}
}

func TestHandleConnectionChatPreservesInternalSpacingAndUnicode(t *testing.T) {
	alice := newConnectionTestEnv(t)
	connectTestPlayer(t, alice, "Alice")

	bob := newSharedConnectionTestEnv(
		t, alice.registry, alice.dispatcher, alice.gameWorld, alice.eventBus, "chat-text-bob",
	)
	connectTestPlayer(t, bob, "Bob")
	if got := alice.readResponse(t); got != "EVT ROOM PRESENCE ENTER Bob" {
		t.Fatalf("Alice's Bob-enter event = %q", got)
	}

	const wantEvent = "EVT ROOM CHAT player=Alice message=Hola,  世界 🌍"
	alice.sendRequest(t, "CHAT   ROOM   Hola,  世界 🌍  ")
	if got := alice.readResponse(t); got != wantEvent {
		t.Fatalf("Alice's chat event = %q, want %q", got, wantEvent)
	}
	if got := alice.readResponse(t); got != "OK" {
		t.Fatalf("chat response = %q, want OK", got)
	}
	if got := bob.readResponse(t); got != wantEvent {
		t.Fatalf("Bob's chat event = %q, want %q", got, wantEvent)
	}
}
