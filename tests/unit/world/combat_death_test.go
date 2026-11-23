package internal_test

import (
	"errors"
	"testing"

	"the-answer-protocol/internal/world"
)

func TestJoinedPlayerGetsNextRoundAfterOriginalPlayerDies(t *testing.T) {
	gameWorld := loadTestWorld(t)
	addPlayersAtDocks(t, gameWorld, "Alice", "Bob")
	woundPlayerForCombatDeathTest(t, gameWorld, "Alice")

	combat, _, err := gameWorld.StartCombat("Alice", "Dock Warden")
	if err != nil {
		t.Fatalf("StartCombat() returned an error: %v", err)
	}
	if current := combat.Participants[combat.TurnIndex].ID; current != "Alice" {
		t.Fatalf("initial turn = %q, want Alice", current)
	}

	if _, _, err := gameWorld.RequestCombatHelp("Alice"); err != nil {
		t.Fatalf("RequestCombatHelp() returned an error: %v", err)
	}
	combat, _, err = gameWorld.JoinCombat("Bob", combat.ID)
	if err != nil {
		t.Fatalf("JoinCombat(Bob) returned an error: %v", err)
	}
	bob := combatParticipantByID(t, combat, "Bob")
	if bob.SpeedPoints != 0 {
		t.Fatalf("Bob's speed points after joining = %d, want 0", bob.SpeedPoints)
	}

	if _, err := gameWorld.SkipPlayerCombatTurn(combat.ID, "Alice"); err != nil {
		t.Fatalf("SkipPlayerCombatTurn(Alice) returned an error: %v", err)
	}
	combat, err = gameWorld.GetCombat(combat.ID)
	if err != nil {
		t.Fatalf("GetCombat() after NPC progression returned an error: %v", err)
	}
	if !combat.Active || combat.Round != 2 {
		t.Fatalf("combat after Alice's death = %+v, want active round 2", combat)
	}
	if _, err := gameWorld.GetActiveCombatForPlayer("Alice"); err != nil {
		t.Fatalf("dead Alice's active combat lookup returned an error: %v", err)
	}
	if _, err := gameWorld.MovePlayer("Alice", "north"); !errors.Is(err, world.ErrPlayerAlreadyInCombat) {
		t.Fatalf("MovePlayer(Alice) after death returned %v, want ErrPlayerAlreadyInCombat", err)
	}
	if _, _, err := gameWorld.RequestCombatHelp("Alice"); !errors.Is(err, world.ErrInvalidCombatTurn) {
		t.Fatalf("RequestCombatHelp(Alice) after death returned %v, want ErrInvalidCombatTurn", err)
	}
	assertPlayerRemainsDeadInRoom(t, gameWorld, "Alice", "docks")

	// The NPC acts first on the tie; Bob must then get his first action.
	combat, err = gameWorld.GetCombat(combat.ID)
	if err != nil {
		t.Fatalf("GetCombat() in round 2 returned an error: %v", err)
	}
	if current := combat.Participants[combat.TurnIndex]; current.ID != "Bob" {
		t.Fatalf("round 2 turn = %q, want Bob", current.ID)
	}
	if world.AvailableActions(combat.Participants[combat.TurnIndex]) == 0 {
		t.Fatal("Bob has no available actions in round 2")
	}

	combat, _, _, _, err = gameWorld.FleeCombat(combat.ID, "Bob")
	if err != nil {
		t.Fatalf("FleeCombat(Bob) returned an error: %v", err)
	}
	if combat.Active {
		t.Fatal("combat remains active after Bob fled")
	}
	assertPlayerRespawnedAfterCombat(t, gameWorld, "Alice")
}

func TestGroupMemberJoiningMidCombatGetsNextRound(t *testing.T) {
	gameWorld := loadTestWorld(t)
	addPlayersAtDocks(t, gameWorld, "Alice", "Bob")

	combat, _, err := gameWorld.StartCombat("Alice", "Dock Warden")
	if err != nil {
		t.Fatalf("StartCombat() returned an error: %v", err)
	}

	if _, err := gameWorld.CreateGroup("Alice", "Rangers"); err != nil {
		t.Fatalf("CreateGroup() returned an error: %v", err)
	}
	if _, err := gameWorld.InvitePlayer("Alice", "Bob"); err != nil {
		t.Fatalf("InvitePlayer() returned an error: %v", err)
	}
	if _, _, err := gameWorld.JoinGroup("Bob", "Rangers"); err != nil {
		t.Fatalf("JoinGroup() returned an error: %v", err)
	}
	combat, _, err = gameWorld.JoinCombat("Bob", combat.ID)
	if err != nil {
		t.Fatalf("JoinCombat(Bob) returned an error: %v", err)
	}
	if joined := combatParticipantByID(t, combat, "Bob"); joined.SpeedPoints != 0 {
		t.Fatalf("Bob's speed points after joining = %d, want 0", joined.SpeedPoints)
	}

	if _, err := gameWorld.SkipPlayerCombatTurn(combat.ID, "Alice"); err != nil {
		t.Fatalf("SkipPlayerCombatTurn(Alice) returned an error: %v", err)
	}
	combat, err = gameWorld.GetCombat(combat.ID)
	if err != nil {
		t.Fatalf("GetCombat() after round 1 actions returned an error: %v", err)
	}
	if combat.TurnIndex < 0 {
		t.Fatalf("turn index after round 1 actions = %d, want an active round 2 turn", combat.TurnIndex)
	}

	if combat.Round != 2 {
		t.Fatalf("combat round = %d, want 2", combat.Round)
	}

	// Alice acts first on the speed tie; after she spends her action, Bob
	// should receive the action points accumulated at the start of round 2.
	if current := combat.Participants[combat.TurnIndex]; current.ID != "Alice" {
		t.Fatalf("first turn in round 2 = %q, want Alice", current.ID)
	}
	if _, err := gameWorld.SkipPlayerCombatTurn(combat.ID, "Alice"); err != nil {
		t.Fatalf("SkipPlayerCombatTurn(Alice) in round 2 returned an error: %v", err)
	}
	if combat, err = gameWorld.GetCombat(combat.ID); err != nil {
		t.Fatalf("GetCombat() after Alice's turn returned an error: %v", err)
	}
	if current := combat.Participants[combat.TurnIndex]; current.ID != "Bob" {
		t.Fatalf("turn after Alice in round 2 = %q, want Bob", current.ID)
	}
	if world.AvailableActions(combat.Participants[combat.TurnIndex]) == 0 {
		t.Fatal("group member Bob has no available actions in round 2")
	}
}

func TestNonGroupPlayerWaitsForCombatToEndAfterDeath(t *testing.T) {
	gameWorld := loadTestWorld(t)
	addPlayersAtDocks(t, gameWorld, "Alice", "Bob")
	woundPlayerForCombatDeathTest(t, gameWorld, "Bob")

	combat, _, err := gameWorld.StartCombat("Alice", "Dock Warden")
	if err != nil {
		t.Fatalf("StartCombat() returned an error: %v", err)
	}
	if current := combat.Participants[combat.TurnIndex].ID; current != "Alice" {
		t.Fatalf("initial turn = %q, want Alice", current)
	}

	if _, _, err := gameWorld.RequestCombatHelp("Alice"); err != nil {
		t.Fatalf("RequestCombatHelp() returned an error: %v", err)
	}
	combat, _, err = gameWorld.JoinCombat("Bob", combat.ID)
	if err != nil {
		t.Fatalf("JoinCombat(Bob) returned an error: %v", err)
	}

	if _, err := gameWorld.SkipPlayerCombatTurn(combat.ID, "Alice"); err != nil {
		t.Fatalf("SkipPlayerCombatTurn(Alice) returned an error: %v", err)
	}
	combat, err = gameWorld.GetCombat(combat.ID)
	if err != nil {
		t.Fatalf("GetCombat() after NPC progression returned an error: %v", err)
	}
	if !combat.Active {
		t.Fatal("combat ended while Alice was still alive")
	}
	if _, err := gameWorld.GetActiveCombatForPlayer("Bob"); err != nil {
		t.Fatalf("dead Bob's active combat lookup returned an error: %v", err)
	}
	assertPlayerRemainsDeadInRoom(t, gameWorld, "Bob", "docks")

	if current := combat.Participants[combat.TurnIndex]; !current.IsPlayer || current.ID != "Alice" {
		t.Fatalf("round 2 turn = %+v, want Alice", current)
	}
	combat, _, _, _, err = gameWorld.FleeCombat(combat.ID, "Alice")
	if err != nil {
		t.Fatalf("FleeCombat(Alice) returned an error: %v", err)
	}
	if combat.Active {
		t.Fatal("combat remains active after Alice fled")
	}
	assertPlayerRespawnedAfterCombat(t, gameWorld, "Bob")
}

func addPlayersAtDocks(
	t *testing.T,
	gameWorld *world.World,
	playerNames ...string,
) {
	t.Helper()

	for _, name := range playerNames {
		if _, err := gameWorld.AddPlayer(name, "docks"); err != nil {
			t.Fatalf("AddPlayer(%s) returned an error: %v", name, err)
		}
	}
}

func woundPlayerForCombatDeathTest(
	t *testing.T,
	gameWorld *world.World,
	playerName string,
) {
	t.Helper()

	combat, _, err := gameWorld.StartCombat(playerName, "Dock Warden")
	if err != nil {
		t.Fatalf("StartCombat(%s) returned an error: %v", playerName, err)
	}

	for actions := 0; actions < 100; actions++ {
		combat, err = gameWorld.GetCombat(combat.ID)
		if err != nil {
			t.Fatalf("GetCombat() returned an error: %v", err)
		}

		if combat.TurnIndex == -1 {
			continue
		}

		participant := combat.Participants[combat.TurnIndex]
		if participant.IsPlayer {
			stats, err := gameWorld.GetPlayerCombatStats(playerName)
			if err != nil {
				t.Fatalf("GetPlayerCombatStats(%s) returned an error: %v", playerName, err)
			}
			if stats.HP <= 5 {
				if _, _, _, _, err := gameWorld.FleeCombat(combat.ID, playerName); err != nil {
					t.Fatalf("FleeCombat(%s) returned an error: %v", playerName, err)
				}
				return
			}
			if _, err := gameWorld.SkipPlayerCombatTurn(combat.ID, playerName); err != nil {
				t.Fatalf("SkipPlayerCombatTurn(%s) returned an error: %v", playerName, err)
			}
			continue
		}

	}

	t.Fatalf("could not wound %s enough to test combat death", playerName)
}

func combatParticipantByID(
	t *testing.T,
	combat *world.Combat,
	participantID string,
) world.CombatParticipant {
	t.Helper()

	for _, participant := range combat.Participants {
		if participant.ID == participantID {
			return participant
		}
	}
	t.Fatalf("combat participant %q not found", participantID)
	return world.CombatParticipant{}
}

func assertPlayerRemainsDeadInRoom(
	t *testing.T,
	gameWorld *world.World,
	playerName string,
	wantRoom string,
) {
	t.Helper()

	roomID, err := gameWorld.GetPlayerRoom(playerName)
	if err != nil {
		t.Fatalf("GetPlayerRoom(%s) returned an error: %v", playerName, err)
	}
	if roomID != wantRoom {
		t.Fatalf("%s's room after death = %q, want %q", playerName, roomID, wantRoom)
	}

	stats, err := gameWorld.GetPlayerCombatStats(playerName)
	if err != nil {
		t.Fatalf("GetPlayerCombatStats(%s) returned an error: %v", playerName, err)
	}
	if stats.HP != 0 {
		t.Fatalf("%s's HP after death = %d, want 0 until combat ends", playerName, stats.HP)
	}
}

func assertPlayerRespawnedAfterCombat(
	t *testing.T,
	gameWorld *world.World,
	playerName string,
) {
	t.Helper()

	roomID, err := gameWorld.GetPlayerRoom(playerName)
	if err != nil {
		t.Fatalf("GetPlayerRoom(%s) returned an error: %v", playerName, err)
	}
	if roomID != world.InitialRoomID {
		t.Fatalf("%s's room after combat = %q, want %q", playerName, roomID, world.InitialRoomID)
	}

	stats, err := gameWorld.GetPlayerCombatStats(playerName)
	if err != nil {
		t.Fatalf("GetPlayerCombatStats(%s) returned an error: %v", playerName, err)
	}
	if stats.HP != stats.MaxHP/4 {
		t.Fatalf("%s's HP after combat = %d, want %d", playerName, stats.HP, stats.MaxHP/4)
	}
}
