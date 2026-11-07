package internal_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
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

func TestHandleConnectionCombatHelpAllowsAnotherPlayerToJoin(t *testing.T) {
	alice := newConnectionTestEnv(t)
	connectTestPlayer(t, alice, "Alice")

	carol := newSharedConnectionTestEnv(
		t, alice.registry, alice.dispatcher, alice.gameWorld, alice.eventBus, "combat-help-carol",
	)
	connectTestPlayer(t, carol, "Carol")
	if got := alice.readResponse(t); got != "EVT ROOM PRESENCE ENTER Carol" {
		t.Fatalf("Alice's Carol-enter event = %q", got)
	}

	alice.sendRequest(t, "MOVE south")
	if got := alice.readResponse(t); got != "OK room=docks" {
		t.Fatalf("Alice's move to docks = %q", got)
	}
	if got := carol.readResponse(t); got != "EVT ROOM PRESENCE LEAVE Alice" {
		t.Fatalf("Carol's Alice-leave event = %q", got)
	}
	carol.sendRequest(t, "MOVE south")
	if got := carol.readResponse(t); got != "OK room=docks" {
		t.Fatalf("Carol's move to docks = %q", got)
	}
	if got := alice.readResponse(t); got != "EVT ROOM PRESENCE ENTER Carol" {
		t.Fatalf("Alice's Carol-enter event = %q", got)
	}

	alice.sendRequest(t, "ATTACK Dock Warden")
	if got := alice.readResponse(t); !strings.HasPrefix(got, "EVT COMBAT STARTED combat=") {
		t.Fatalf("combat-start event = %q", got)
	}
	if got := alice.readResponse(t); !strings.HasPrefix(got, "EVT COMBAT TURN ") {
		t.Fatalf("combat-turn event = %q", got)
	}
	attackResponse := alice.readResponse(t)
	if !strings.HasPrefix(attackResponse, "OK ") {
		t.Fatalf("attack response = %q, want OK JSON", attackResponse)
	}

	var attackResult struct {
		CombatID string `json:"combat_id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(attackResponse, "OK ")), &attackResult); err != nil {
		t.Fatalf("decode attack response: %v; response: %q", err, attackResponse)
	}
	if attackResult.CombatID == "" {
		t.Fatalf("attack response has an empty combat ID: %q", attackResponse)
	}

	alice.sendRequest(t, "COMBAT HELP")
	if got := alice.readResponse(t); got != "OK help_requested=true combat="+attackResult.CombatID {
		t.Fatalf("combat-help response = %q", got)
	}
	if got := carol.readResponse(t); got != "EVT COMBAT HELP_REQUESTED player=Alice combat="+attackResult.CombatID+" room=docks" {
		t.Fatalf("combat-help notification = %q", got)
	}

	combat, err := alice.gameWorld.GetCombat(attackResult.CombatID)
	if err != nil {
		t.Fatalf("GetCombat() returned an error: %v", err)
	}
	if !combat.HelpRequested {
		t.Fatal("combat state does not record the help request")
	}

	carol.sendRequest(t, "COMBAT JOIN "+attackResult.CombatID)
	wantJoinEvent := "EVT COMBAT PLAYER_JOINED player=Carol combat=" + attackResult.CombatID
	if got := alice.readResponse(t); got != wantJoinEvent {
		t.Fatalf("Alice's join notification = %q, want %q", got, wantJoinEvent)
	}
	if got := carol.readResponse(t); got != wantJoinEvent {
		t.Fatalf("Carol's join notification = %q, want %q", got, wantJoinEvent)
	}
	if got := carol.readResponse(t); got != "OK joined=true combat="+attackResult.CombatID+" target_hp=140" {
		t.Fatalf("combat-join response = %q", got)
	}

	combat, err = alice.gameWorld.GetCombat(attackResult.CombatID)
	if err != nil {
		t.Fatalf("GetCombat() after join returned an error: %v", err)
	}
	joined := false
	for _, participant := range combat.Participants {
		if participant.IsPlayer && participant.ID == "Carol" {
			joined = true
			break
		}
	}
	if !joined {
		t.Fatal("Carol is not a participant after joining the combat")
	}
}

func TestHandleConnectionCombatContinuesWhenParticipantDisconnects(t *testing.T) {
	alice, bob := newGroupAtDocks(t)

	alice.sendRequest(t, "ATTACK Dock Warden")
	aliceStarted := alice.readResponse(t)
	if !strings.HasPrefix(aliceStarted, "EVT COMBAT STARTED combat=") {
		t.Fatalf("combat-start event = %q", aliceStarted)
	}
	if got := bob.readResponse(t); got != aliceStarted {
		t.Fatalf("Bob's combat-start event = %q, want %q", got, aliceStarted)
	}
	aliceTurn := alice.readResponse(t)
	if !strings.HasPrefix(aliceTurn, "EVT COMBAT TURN ") {
		t.Fatalf("combat-turn event = %q", aliceTurn)
	}
	if got := bob.readResponse(t); got != aliceTurn {
		t.Fatalf("Bob's combat-turn event = %q, want %q", got, aliceTurn)
	}
	attackResponse := alice.readResponse(t)
	if !strings.HasPrefix(attackResponse, "OK ") {
		t.Fatalf("combat-start response = %q", attackResponse)
	}

	var result struct {
		CombatID string `json:"combat_id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(attackResponse, "OK ")), &result); err != nil {
		t.Fatalf("decode combat-start response: %v; response: %q", err, attackResponse)
	}
	combat, err := alice.gameWorld.GetCombat(result.CombatID)
	if err != nil {
		t.Fatalf("GetCombat(%q) returned an error: %v", result.CombatID, err)
	}
	currentPlayer := combat.Participants[combat.TurnIndex].ID
	disconnected, remaining := alice, bob
	disconnectedName, remainingName := "Alice", "Bob"
	if currentPlayer == "Bob" {
		disconnected, remaining = bob, alice
		disconnectedName, remainingName = "Bob", "Alice"
	} else if currentPlayer != "Alice" {
		t.Fatalf("combat turn participant = %q, want a player", currentPlayer)
	}

	if err := disconnected.client.Close(); err != nil {
		t.Fatalf("close %s's connection: %v", disconnectedName, err)
	}
	if got := remaining.readResponse(t); got != "EVT ROOM PRESENCE LEAVE "+disconnectedName {
		t.Fatalf("%s's player-leave event = %q", remainingName, got)
	}
	turnEvent := remaining.readResponse(t)
	if !strings.HasPrefix(turnEvent, "EVT COMBAT TURN participant_type=PLAYER participant="+remainingName+" ") {
		t.Fatalf("%s's reassigned combat turn = %q", remainingName, turnEvent)
	}

	combat, err = remaining.gameWorld.GetCombat(result.CombatID)
	if err != nil {
		t.Fatalf("GetCombat() after disconnect returned an error: %v", err)
	}
	if !combat.Active || combat.Participants[combat.TurnIndex].ID != remainingName {
		t.Fatalf("combat after %s disconnects = %+v, want active turn for %s", disconnectedName, combat, remainingName)
	}
	if _, err := remaining.gameWorld.GetActiveCombatForPlayer(remainingName); err != nil {
		t.Fatalf("%s cannot continue combat: %v", remainingName, err)
	}

	remaining.sendRequest(t, "COMBAT FLEE")
	if got := remaining.readResponse(t); !strings.HasPrefix(got, "OK fled=true ") {
		t.Fatalf("combat-flee response = %q", got)
	}
	combat, err = remaining.gameWorld.GetCombat(result.CombatID)
	if err != nil {
		t.Fatalf("GetCombat() after flee returned an error: %v", err)
	}
	if combat.Active {
		t.Fatal("combat remains active after the only connected participant flees")
	}
}

func TestHandleConnectionCombatEndsWhenAllParticipantsDisconnect(t *testing.T) {
	alice := newConnectionTestEnv(t)
	connectTestPlayer(t, alice, "Alice")
	alice.sendRequest(t, "MOVE south")
	if got := alice.readResponse(t); got != "OK room=docks" {
		t.Fatalf("Alice's move to docks = %q", got)
	}

	alice.sendRequest(t, "ATTACK Dock Warden")
	if got := alice.readResponse(t); !strings.HasPrefix(got, "EVT COMBAT STARTED combat=") {
		t.Fatalf("combat-start event = %q", got)
	}
	if got := alice.readResponse(t); !strings.HasPrefix(got, "EVT COMBAT TURN ") {
		t.Fatalf("combat-turn event = %q", got)
	}
	attackResponse := alice.readResponse(t)
	var result struct {
		CombatID string `json:"combat_id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(attackResponse, "OK ")), &result); err != nil {
		t.Fatalf("decode combat-start response: %v; response: %q", err, attackResponse)
	}

	if err := alice.client.Close(); err != nil {
		t.Fatalf("close Alice's connection: %v", err)
	}
	deadline := time.Now().Add(connectionTestTimeout)
	for time.Now().Before(deadline) {
		combat, err := alice.gameWorld.GetCombat(result.CombatID)
		if err == nil && !combat.Active {
			assertPlayerRemoved(t, alice, "Alice")
			if _, err := alice.gameWorld.GetActiveCombatForPlayer("Alice"); err == nil {
				t.Fatal("disconnected Alice still has an active combat")
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("combat remained active after its only participant disconnected")
}

func TestHandleConnectionCombatVictoryAwardsReward(t *testing.T) {
	alice := newConnectionTestEnv(t)
	connectTestPlayer(t, alice, "Alice")
	for _, step := range []struct {
		direction string
		room      string
	}{
		{"north", "market_street"},
		{"north", "town_gate"},
		{"north", "forest_edge"},
	} {
		alice.sendRequest(t, "MOVE "+step.direction)
		if got := alice.readResponse(t); got != "OK room="+step.room {
			t.Fatalf("move to %s response = %q", step.room, got)
		}
	}

	alice.sendRequest(t, "ATTACK Dire Wolf")
	if got := alice.readResponse(t); !strings.HasPrefix(got, "EVT COMBAT STARTED combat=") {
		t.Fatalf("combat-start event = %q", got)
	}
	if got := alice.readResponse(t); !strings.HasPrefix(got, "EVT COMBAT TURN ") {
		t.Fatalf("combat-turn event = %q", got)
	}
	startResponse := alice.readResponse(t)
	var start struct {
		CombatID string `json:"combat_id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(startResponse, "OK ")), &start); err != nil {
		t.Fatalf("decode combat-start response: %v; response: %q", err, startResponse)
	}

	for attacks := 0; attacks < 20; attacks++ {
		combat, err := alice.gameWorld.GetCombat(start.CombatID)
		if err != nil {
			t.Fatalf("GetCombat() returned an error: %v", err)
		}
		if !combat.Active {
			break
		}
		if combat.Participants[combat.TurnIndex].ID != "Alice" {
			t.Fatalf("combat turn = %+v, want Alice to act", combat.Participants[combat.TurnIndex])
		}

		alice.sendRequest(t, "COMBAT ATTACK Dire Wolf")
		response := readUntilResponse(t, alice)
		if !strings.HasPrefix(response, "OK target=dire_wolf ") {
			t.Fatalf("combat attack response = %q", response)
		}
	}

	combat, err := alice.gameWorld.GetCombat(start.CombatID)
	if err != nil {
		t.Fatalf("GetCombat() after attacks returned an error: %v", err)
	}
	if combat.Active {
		t.Fatal("combat remains active after the Dire Wolf should have been defeated")
	}
	wolfDefeated := false
	for _, participant := range combat.Participants {
		if participant.ID == "dire_wolf" && !participant.Alive {
			wolfDefeated = true
		}
	}
	if !wolfDefeated {
		t.Fatalf("Dire Wolf was not defeated: %+v", combat.Participants)
	}
	inventory, err := alice.gameWorld.GetInventory("Alice")
	if err != nil {
		t.Fatalf("GetInventory() returned an error: %v", err)
	}
	herbCount := 0
	for _, item := range inventory {
		if item.ID == "item.herbs" {
			herbCount++
		}
	}
	if herbCount != 1 {
		t.Fatalf("Dire Wolf reward quantity = %d, want 1", herbCount)
	}
}

func TestHandleConnectionCombatDefeatRespawnsPlayer(t *testing.T) {
	alice := newConnectionTestEnv(t)
	connectTestPlayer(t, alice, "Alice")
	alice.sendRequest(t, "MOVE south")
	if got := alice.readResponse(t); got != "OK room=docks" {
		t.Fatalf("Alice's move to docks = %q", got)
	}

	alice.sendRequest(t, "ATTACK Dock Warden")
	if got := alice.readResponse(t); !strings.HasPrefix(got, "EVT COMBAT STARTED combat=") {
		t.Fatalf("combat-start event = %q", got)
	}
	if got := alice.readResponse(t); !strings.HasPrefix(got, "EVT COMBAT TURN ") {
		t.Fatalf("combat-turn event = %q", got)
	}
	startResponse := alice.readResponse(t)
	var start struct {
		CombatID string `json:"combat_id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(startResponse, "OK ")), &start); err != nil {
		t.Fatalf("decode combat-start response: %v; response: %q", err, startResponse)
	}

	for attacks := 0; attacks < 20; attacks++ {
		combat, err := alice.gameWorld.GetCombat(start.CombatID)
		if err != nil {
			t.Fatalf("GetCombat() returned an error: %v", err)
		}
		if !combat.Active {
			break
		}
		if combat.Participants[combat.TurnIndex].ID != "Alice" {
			t.Fatalf("combat turn = %+v, want Alice to act", combat.Participants[combat.TurnIndex])
		}

		alice.sendRequest(t, "COMBAT ATTACK Dock Warden")
		response := readUntilResponse(t, alice)
		if !strings.HasPrefix(response, "OK target=dock_warden ") {
			t.Fatalf("combat attack response = %q", response)
		}
	}

	combat, err := alice.gameWorld.GetCombat(start.CombatID)
	if err != nil {
		t.Fatalf("GetCombat() after attacks returned an error: %v", err)
	}
	if combat.Active {
		t.Fatal("combat remains active after Alice should have been defeated")
	}
	if got, err := alice.gameWorld.GetPlayerRoom("Alice"); err != nil || got != "harbor_square" {
		t.Fatalf("Alice's room after defeat = (%q, %v), want harbor_square", got, err)
	}
	if _, err := alice.gameWorld.GetActiveCombatForPlayer("Alice"); err == nil {
		t.Fatal("defeated Alice still has an active combat")
	}
}

func TestHandleConnectionCombatFleeEndsSoloEncounter(t *testing.T) {
	alice := newConnectionTestEnv(t)
	connectTestPlayer(t, alice, "Alice")
	alice.sendRequest(t, "MOVE south")
	if got := alice.readResponse(t); got != "OK room=docks" {
		t.Fatalf("Alice's move to docks = %q", got)
	}

	alice.sendRequest(t, "ATTACK Dock Warden")
	if got := alice.readResponse(t); !strings.HasPrefix(got, "EVT COMBAT STARTED combat=") {
		t.Fatalf("combat-start event = %q", got)
	}
	if got := alice.readResponse(t); !strings.HasPrefix(got, "EVT COMBAT TURN ") {
		t.Fatalf("combat-turn event = %q", got)
	}
	startResponse := alice.readResponse(t)
	var start struct {
		CombatID string `json:"combat_id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(startResponse, "OK ")), &start); err != nil {
		t.Fatalf("decode combat-start response: %v; response: %q", err, startResponse)
	}

	alice.sendRequest(t, "COMBAT FLEE")
	fleeEvent := readUntilResponse(t, alice)
	if !strings.HasPrefix(fleeEvent, "OK fled=true damage=") ||
		!strings.Contains(fleeEvent, " combat="+start.CombatID) {
		t.Fatalf("combat-flee response = %q", fleeEvent)
	}

	combat, err := alice.gameWorld.GetCombat(start.CombatID)
	if err != nil {
		t.Fatalf("GetCombat() after flee returned an error: %v", err)
	}
	if combat.Active {
		t.Fatal("combat remains active after the only participant flees")
	}
	if got, err := alice.gameWorld.GetPlayerRoom("Alice"); err != nil || got != "docks" {
		t.Fatalf("Alice's room after fleeing = (%q, %v), want docks", got, err)
	}
}

func newGroupAtDocks(t *testing.T) (*connectionTestEnv, *connectionTestEnv) {
	t.Helper()

	alice := newConnectionTestEnv(t)
	connectTestPlayer(t, alice, "Alice")
	bob := newSharedConnectionTestEnv(
		t, alice.registry, alice.dispatcher, alice.gameWorld, alice.eventBus, "combat-disconnect-bob",
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
		t.Fatalf("Alice's move to docks response = %q", got)
	}
	if got := bob.readResponse(t); got != "EVT ROOM PRESENCE LEAVE Alice" {
		t.Fatalf("Bob's Alice-leave event = %q", got)
	}

	bob.sendRequest(t, "MOVE south")
	if got := bob.readResponse(t); got != "OK room=docks" {
		t.Fatalf("Bob's move to docks response = %q", got)
	}
	if got := alice.readResponse(t); got != "EVT ROOM PRESENCE ENTER Bob" {
		t.Fatalf("Alice's Bob-enter event = %q", got)
	}
	return alice, bob
}

func readUntilResponse(t *testing.T, env *connectionTestEnv) string {
	t.Helper()

	for {
		line := env.readResponse(t)
		if strings.HasPrefix(line, "OK ") || strings.HasPrefix(line, "ERR ") {
			return line
		}
	}
}
