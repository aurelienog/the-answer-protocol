package internal_test

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHandleConnectionGroupStartsCooperativeCombat(t *testing.T) {
	alice := newConnectionTestEnv(t)
	connectTestPlayer(t, alice, "Alice")

	bob := newSharedConnectionTestEnv(
		t, alice.registry, alice.dispatcher, alice.gameWorld, alice.eventBus, "combat-bob",
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

	alice.sendRequest(t, "MOVE south")
	if got := alice.readResponse(t); got != "OK room=docks" {
		t.Fatalf("Alice's move to the docks = %q", got)
	}
	if got := bob.readResponse(t); got != "EVT ROOM PRESENCE LEAVE Alice" {
		t.Fatalf("Bob's Alice-leave event = %q", got)
	}
	bob.sendRequest(t, "MOVE south")
	if got := bob.readResponse(t); got != "OK room=docks" {
		t.Fatalf("Bob's move to the docks = %q", got)
	}
	if got := alice.readResponse(t); got != "EVT ROOM PRESENCE ENTER Bob" {
		t.Fatalf("Alice's Bob-enter event = %q", got)
	}

	alice.sendRequest(t, "ATTACK Dock Warden")
	aliceStarted := alice.readResponse(t)
	bobStarted := bob.readResponse(t)
	if !strings.HasPrefix(aliceStarted, "EVT COMBAT STARTED combat=combat:") {
		t.Fatalf("Alice's combat-start event = %q", aliceStarted)
	}
	if bobStarted != aliceStarted {
		t.Fatalf("combat-start events differ: Alice=%q Bob=%q", aliceStarted, bobStarted)
	}
	if !strings.Contains(aliceStarted, " room=docks participants=Alice,Bob") {
		t.Errorf("combat-start event does not include both group members: %q", aliceStarted)
	}

	aliceTurn := alice.readResponse(t)
	bobTurn := bob.readResponse(t)
	if !strings.HasPrefix(aliceTurn, "EVT COMBAT TURN participant_type=") {
		t.Fatalf("Alice's combat-turn event = %q", aliceTurn)
	}
	if bobTurn != aliceTurn {
		t.Fatalf("combat-turn events differ: Alice=%q Bob=%q", aliceTurn, bobTurn)
	}

	attackResponse := alice.readResponse(t)
	if !strings.HasPrefix(attackResponse, "OK ") {
		t.Fatalf("attack response = %q, want OK JSON", attackResponse)
	}
	var response struct {
		CombatID string `json:"combat_id"`
		Status   string `json:"status"`
		TargetHP int    `json:"target_hp"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(attackResponse, "OK ")), &response); err != nil {
		t.Fatalf("decode attack response: %v; response: %q", err, attackResponse)
	}
	if response.CombatID == "" || response.Status != "combat" || response.TargetHP <= 0 {
		t.Fatalf("unexpected attack response payload: %+v", response)
	}

	combat, err := alice.gameWorld.GetCombat(response.CombatID)
	if err != nil {
		t.Fatalf("GetCombat(%q) returned an error: %v", response.CombatID, err)
	}
	if !combat.Active {
		t.Fatal("combat is not active")
	}
	participants := make(map[string]bool)
	for _, participant := range combat.Participants {
		if participant.IsPlayer {
			participants[participant.ID] = true
		}
	}
	if !participants["Alice"] || !participants["Bob"] {
		t.Errorf("combat player participants = %v, want Alice and Bob", participants)
	}
	if _, err := alice.gameWorld.GetActiveCombatForPlayer("Alice"); err != nil {
		t.Errorf("Alice has no active combat: %v", err)
	}
	if _, err := alice.gameWorld.GetActiveCombatForPlayer("Bob"); err != nil {
		t.Errorf("Bob has no active combat: %v", err)
	}
}
