package server_test

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"the-answer-protocol/internal/events"
	"the-answer-protocol/internal/server"
	"the-answer-protocol/internal/world"
)

func TestNPCRespawnManagerRunsLatestScheduledCallback(t *testing.T) {
	callbacks := make(chan world.NPCRespawnScheduled, 2)
	manager := server.NewNPCRespawnManager(func(event world.NPCRespawnScheduled) {
		callbacks <- event
	})
	defer manager.Stop()

	manager.Handle(world.NPCRespawnScheduled{
		NPCID:     "npc-1",
		RespawnAt: time.Now().Add(30 * time.Millisecond),
	})
	latest := world.NPCRespawnScheduled{
		NPCID:     "npc-1",
		RespawnAt: time.Now().Add(80 * time.Millisecond),
	}
	manager.Handle(latest)

	select {
	case got := <-callbacks:
		if !got.RespawnAt.Equal(latest.RespawnAt) {
			t.Fatalf("scheduled callback = %+v, want latest deadline %+v", got, latest)
		}
	case <-time.After(time.Second):
		t.Fatal("NPC respawn callback did not run")
	}
}

func TestNPCRespawnManagerStopCancelsPendingCallbacks(t *testing.T) {
	callbacks := make(chan world.NPCRespawnScheduled, 1)
	manager := server.NewNPCRespawnManager(func(event world.NPCRespawnScheduled) {
		callbacks <- event
	})
	manager.Handle(world.NPCRespawnScheduled{
		NPCID:     "npc-1",
		RespawnAt: time.Now().Add(30 * time.Millisecond),
	})
	manager.Stop()

	select {
	case event := <-callbacks:
		t.Fatalf("callback ran after manager stopped: %+v", event)
	case <-time.After(60 * time.Millisecond):
	}
}

func TestNPCRespawnManagerRestoresNPCAndPublishesEvent(t *testing.T) {
	worldPath := worldWithRespawnSeconds(t, "dire_wolf", 1)
	gameWorld, err := world.Load(worldPath)
	if err != nil {
		t.Fatalf("load test world: %v", err)
	}
	if _, err := gameWorld.AddPlayer("Alice", "forest_edge"); err != nil {
		t.Fatalf("AddPlayer(Alice): %v", err)
	}

	eventBus := events.NewBus()
	respawned := make(chan world.NPCRespawned, 1)
	eventBus.Subscribe(func(event any) {
		if value, ok := event.(world.NPCRespawned); ok {
			respawned <- value
		}
	})
	manager := server.NewNPCRespawnManager(
		server.NewNPCRespawnHandler(gameWorld, eventBus, slog.Default()),
	)
	eventBus.Subscribe(manager.Handle)
	defer manager.Stop()

	combat, combatEvents, err := gameWorld.StartCombat("Alice", "Dire Wolf")
	if err != nil {
		t.Fatalf("StartCombat(): %v", err)
	}
	publishEvents := func(events []any) {
		for _, event := range events {
			eventBus.Publish(event)
		}
	}
	publishEvents(combatEvents)

	for actions := 0; actions < 100; actions++ {
		combat, err = gameWorld.GetCombat(combat.ID)
		if err != nil {
			t.Fatalf("GetCombat(): %v", err)
		}
		if !combat.Active {
			break
		}

		participant := combat.Participants[combat.TurnIndex]
		if !participant.IsPlayer {
			continue
		}
		_, _, combatEvents, err = gameWorld.AttackCombatParticipant(
			combat.ID,
			participant.ID,
			"dire_wolf",
		)
		if err != nil {
			t.Fatalf("AttackCombatParticipant(): %v", err)
		}
		publishEvents(combatEvents)
		if actions == 99 {
			t.Fatal("combat did not end within 100 actions")
		}
	}

	select {
	case event := <-respawned:
		if event.NPCID != "dire_wolf" || event.RoomID != "forest_edge" {
			t.Fatalf("NPC respawn event = %+v", event)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("NPC did not respawn after its configured delay")
	}

	npcs, err := gameWorld.GetNPCsInRoom("forest_edge")
	if err != nil {
		t.Fatalf("GetNPCsInRoom(forest_edge): %v", err)
	}
	for _, npc := range npcs {
		if npc.ID == "dire_wolf" &&
			(npc.CombatStats.HP != npc.CombatStats.MaxHP || !npc.RespawnAt.IsZero()) {
			t.Fatalf("NPC after respawn = %+v, want full health and no pending deadline", npc)
		}
	}
}

func worldWithRespawnSeconds(
	t *testing.T,
	npcID string,
	seconds int,
) string {
	t.Helper()

	data, err := os.ReadFile("../../../data/world.json")
	if err != nil {
		t.Fatalf("read test world: %v", err)
	}

	var definition map[string]json.RawMessage
	if err := json.Unmarshal(data, &definition); err != nil {
		t.Fatalf("decode test world: %v", err)
	}
	var npcs map[string]json.RawMessage
	if err := json.Unmarshal(definition["npcs"], &npcs); err != nil {
		t.Fatalf("decode test NPCs: %v", err)
	}
	var npc map[string]json.RawMessage
	if err := json.Unmarshal(npcs[npcID], &npc); err != nil {
		t.Fatalf("decode NPC %q: %v", npcID, err)
	}
	respawnSeconds, err := json.Marshal(seconds)
	if err != nil {
		t.Fatalf("encode respawn duration: %v", err)
	}
	npc["respawn_seconds"] = respawnSeconds
	npcs[npcID], err = json.Marshal(npc)
	if err != nil {
		t.Fatalf("encode NPC %q: %v", npcID, err)
	}
	definition["npcs"], err = json.Marshal(npcs)
	if err != nil {
		t.Fatalf("encode test NPCs: %v", err)
	}
	data, err = json.Marshal(definition)
	if err != nil {
		t.Fatalf("encode test world: %v", err)
	}

	path := filepath.Join(t.TempDir(), "world.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write test world: %v", err)
	}
	return path
}
