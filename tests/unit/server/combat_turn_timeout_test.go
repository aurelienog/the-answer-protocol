package server_test

import (
	"testing"
	"time"

	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/events"
	"the-answer-protocol/internal/server"
	"the-answer-protocol/internal/world"
)

func TestCombatTurnTimeoutContinuesAfterPlayerJoinsMidRound(t *testing.T) {
	gameWorld, err := world.Load("../../../data/world.json")
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	for _, name := range []string{"Alice", "Bob"} {
		if _, err := gameWorld.AddPlayer(name, "forest_edge"); err != nil {
			t.Fatalf("AddPlayer(%s): %v", name, err)
		}
	}

	combat, _, err := gameWorld.StartCombat("Alice", "Dire Wolf")
	if err != nil {
		t.Fatalf("StartCombat(): %v", err)
	}
	combat, turn, err := gameWorld.InitializeCombatTurns(combat.ID)
	if err != nil {
		t.Fatalf("InitializeCombatTurns(): %v", err)
	}
	if turn == nil || !turn.IsPlayer || turn.ParticipantID != "Alice" {
		t.Fatalf("initial combat turn = %+v, want Alice's turn", turn)
	}
	if _, _, err := gameWorld.RequestCombatHelp("Alice"); err != nil {
		t.Fatalf("RequestCombatHelp(): %v", err)
	}
	if _, _, err := gameWorld.JoinCombat("Bob", combat.ID); err != nil {
		t.Fatalf("JoinCombat(Bob): %v", err)
	}

	eventBus := events.NewBus()
	manager := server.NewCombatTurnTimeoutManager(
		100*time.Millisecond,
		commands.NewCombatTurnTimeoutHandler(gameWorld, eventBus),
	)
	defer manager.Stop()
	eventBus.Subscribe(manager.Handle)
	manager.Handle(*turn)

	_, err = commands.NewCombatCommand().Execute(&commands.Context{
		PlayerName: "Alice",
		RawArgs:    []string{"ATTACK", "Dire Wolf"},
		World:      gameWorld,
		Events:     eventBus,
	})
	if err != nil {
		t.Fatalf("Alice's attack: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		combat, err = gameWorld.GetCombat(combat.ID)
		if err != nil {
			t.Fatalf("GetCombat(): %v", err)
		}
		if combat.Round >= 3 {
			return
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatalf("combat did not advance after the joined player's timeout: %+v", combat)
}
