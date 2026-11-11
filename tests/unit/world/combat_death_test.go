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
	combat, _, err = gameWorld.InitializeCombatTurns(combat.ID)
	if err != nil {
		t.Fatalf("InitializeCombatTurns() returned an error: %v", err)
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
	combat, _, _, _, death, err := gameWorld.ExecuteNPCCombatTurn(combat.ID)
	if err != nil {
		t.Fatalf("ExecuteNPCCombatTurn() returned an error: %v", err)
	}
	if death == nil || death.ParticipantID != "Alice" {
		t.Fatalf("NPC death event = %+v, want Alice", death)
	}
	if !combat.Active || combat.Round != 1 {
		t.Fatalf("combat after Alice's death = %+v, want active round 1", combat)
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
	if _, err := gameWorld.EndCombat(combat.ID); !errors.Is(err, world.ErrInvalidWorldState) {
		t.Fatalf("EndCombat() while Bob is alive returned %v, want ErrInvalidWorldState", err)
	}
	assertPlayerRemainsDeadInRoom(t, gameWorld, "Alice", "docks")

	combat, _, err = gameWorld.StartCombatRound(combat.ID)
	if err != nil {
		t.Fatalf("StartCombatRound() returned an error: %v", err)
	}
	if combat.Round != 2 {
		t.Fatalf("combat round = %d, want 2", combat.Round)
	}

	// The NPC acts first on the tie; Bob must then get his first action.
	combat, _, _, _, _, err = gameWorld.ExecuteNPCCombatTurn(combat.ID)
	if err != nil {
		t.Fatalf("ExecuteNPCCombatTurn() in round 2 returned an error: %v", err)
	}
	if current := combat.Participants[combat.TurnIndex]; current.ID != "Bob" {
		t.Fatalf("round 2 turn = %q, want Bob", current.ID)
	}
	if world.AvailableActions(combat.Participants[combat.TurnIndex]) == 0 {
		t.Fatal("Bob has no available actions in round 2")
	}

	if _, _, err := gameWorld.FleeCombat(combat.ID, "Bob"); err != nil {
		t.Fatalf("FleeCombat(Bob) returned an error: %v", err)
	}
	combat, err = gameWorld.EndCombat(combat.ID)
	if err != nil {
		t.Fatalf("EndCombat() after Bob fled returned an error: %v", err)
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
	combat, _, err = gameWorld.InitializeCombatTurns(combat.ID)
	if err != nil {
		t.Fatalf("InitializeCombatTurns() returned an error: %v", err)
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
	combat, _, _, _, _, err = gameWorld.ExecuteNPCCombatTurn(combat.ID)
	if err != nil {
		t.Fatalf("ExecuteNPCCombatTurn() returned an error: %v", err)
	}
	if combat.TurnIndex != -1 {
		t.Fatalf("turn index after round 1 actions = %d, want -1", combat.TurnIndex)
	}

	combat, _, err = gameWorld.StartCombatRound(combat.ID)
	if err != nil {
		t.Fatalf("StartCombatRound() returned an error: %v", err)
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
	if current := combat.Participants[combat.TurnIndex]; current.IsPlayer {
		t.Fatalf("turn after Alice in round 2 = %q, want NPC", current.ID)
	}
	combat, _, _, _, _, err = gameWorld.ExecuteNPCCombatTurn(combat.ID)
	if err != nil {
		t.Fatalf("ExecuteNPCCombatTurn() in round 2 returned an error: %v", err)
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
	combat, _, err = gameWorld.InitializeCombatTurns(combat.ID)
	if err != nil {
		t.Fatalf("InitializeCombatTurns() returned an error: %v", err)
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
	combat, _, _, _, death, err := gameWorld.ExecuteNPCCombatTurn(combat.ID)
	if err != nil {
		t.Fatalf("ExecuteNPCCombatTurn() returned an error: %v", err)
	}
	if death == nil || death.ParticipantID != "Bob" {
		t.Fatalf("NPC death event = %+v, want Bob", death)
	}
	if !combat.Active {
		t.Fatal("combat ended while Alice was still alive")
	}
	if _, err := gameWorld.GetActiveCombatForPlayer("Bob"); err != nil {
		t.Fatalf("dead Bob's active combat lookup returned an error: %v", err)
	}
	if _, err := gameWorld.EndCombat(combat.ID); !errors.Is(err, world.ErrInvalidWorldState) {
		t.Fatalf("EndCombat() while Alice is alive returned %v, want ErrInvalidWorldState", err)
	}
	assertPlayerRemainsDeadInRoom(t, gameWorld, "Bob", "docks")

	combat, _, err = gameWorld.StartCombatRound(combat.ID)
	if err != nil {
		t.Fatalf("StartCombatRound() returned an error: %v", err)
	}
	if current := combat.Participants[combat.TurnIndex]; !current.IsPlayer || current.ID != "Alice" {
		t.Fatalf("round 2 turn = %+v, want Alice", current)
	}
	if _, _, err := gameWorld.FleeCombat(combat.ID, "Alice"); err != nil {
		t.Fatalf("FleeCombat(Alice) returned an error: %v", err)
	}
	combat, err = gameWorld.EndCombat(combat.ID)
	if err != nil {
		t.Fatalf("EndCombat() after Alice fled returned an error: %v", err)
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
	combat, _, err = gameWorld.InitializeCombatTurns(combat.ID)
	if err != nil {
		t.Fatalf("InitializeCombatTurns() returned an error: %v", err)
	}

	for actions := 0; actions < 100; actions++ {
		combat, err = gameWorld.GetCombat(combat.ID)
		if err != nil {
			t.Fatalf("GetCombat() returned an error: %v", err)
		}

		if combat.TurnIndex == -1 {
			combat, _, err = gameWorld.StartCombatRound(combat.ID)
			if err != nil {
				t.Fatalf("StartCombatRound() returned an error: %v", err)
			}
			continue
		}

		participant := combat.Participants[combat.TurnIndex]
		if participant.IsPlayer {
			stats, err := gameWorld.GetPlayerCombatStats(playerName)
			if err != nil {
				t.Fatalf("GetPlayerCombatStats(%s) returned an error: %v", playerName, err)
			}
			if stats.HP <= 5 {
				if _, _, err := gameWorld.FleeCombat(combat.ID, playerName); err != nil {
					t.Fatalf("FleeCombat(%s) returned an error: %v", playerName, err)
				}
				if _, err := gameWorld.EndCombat(combat.ID); err != nil {
					t.Fatalf("EndCombat() after fleeing returned an error: %v", err)
				}
				return
			}
			if _, err := gameWorld.SkipPlayerCombatTurn(combat.ID, playerName); err != nil {
				t.Fatalf("SkipPlayerCombatTurn(%s) returned an error: %v", playerName, err)
			}
			continue
		}

		if _, _, _, _, _, err := gameWorld.ExecuteNPCCombatTurn(combat.ID); err != nil {
			t.Fatalf("ExecuteNPCCombatTurn() returned an error: %v", err)
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
