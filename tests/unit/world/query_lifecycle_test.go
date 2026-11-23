package internal_test

import (
	"errors"
	"reflect"
	"testing"

	"the-answer-protocol/internal/world"
)

func TestWorldPlayerAndRoomQueries(t *testing.T) {
	gameWorld := loadTestWorld(t)
	for name, roomID := range map[string]string{
		"Zoe":   "harbor_square",
		"Alice": "harbor_square",
		"Bob":   "market_street",
	} {
		if _, err := gameWorld.AddPlayer(name, roomID); err != nil {
			t.Fatalf("AddPlayer(%s) returned an error: %v", name, err)
		}
	}

	count, err := gameWorld.GetPlayerCount()
	if err != nil || count != 3 {
		t.Fatalf("GetPlayerCount() = (%d, %v), want 3 players", count, err)
	}
	players, err := gameWorld.GetPlayersInRoom("harbor_square")
	if err != nil || !reflect.DeepEqual(players, []string{"Alice", "Zoe"}) {
		t.Fatalf("GetPlayersInRoom(harbor_square) = (%v, %v), want sorted names", players, err)
	}

	item, err := gameWorld.FindRoomItem("harbor_square", "  hempen rope ")
	if err != nil || item.ID != "item.rope" {
		t.Fatalf("FindRoomItem() = (%+v, %v), want item.rope", item, err)
	}
	item, err = gameWorld.FindRoomItem("market_street", "ITEM.POUCH")
	if err != nil || item.ID != "item.pouch" {
		t.Fatalf("FindRoomItem() by ID = (%+v, %v), want item.pouch", item, err)
	}
	if _, err := gameWorld.FindRoomItem("harbor_square", ""); !errors.Is(err, world.ErrInvalidArgument) {
		t.Fatalf("FindRoomItem(empty query) error = %v, want ErrInvalidArgument", err)
	}
	if _, err := gameWorld.FindRoomItem("harbor_square", "missing"); !errors.Is(err, world.ErrItemNotFound) {
		t.Fatalf("FindRoomItem(missing item) error = %v, want ErrItemNotFound", err)
	}

	if _, err := gameWorld.GetPlayersInRoom("missing_room"); !errors.Is(err, world.ErrInvalidWorldState) {
		t.Fatalf("GetPlayersInRoom(missing room) error = %v, want ErrInvalidWorldState", err)
	}
	if _, err := gameWorld.AddPlayer("Alice", "market_street"); !errors.Is(err, world.ErrPlayerAlreadyExists) {
		t.Fatalf("AddPlayer(duplicate) error = %v, want ErrPlayerAlreadyExists", err)
	}
	if _, err := gameWorld.AddPlayer(" ", "market_street"); !errors.Is(err, world.ErrInvalidArgument) {
		t.Fatalf("AddPlayer(empty name) error = %v, want ErrInvalidArgument", err)
	}
	if _, err := gameWorld.AddPlayer("Eve", "missing_room"); !errors.Is(err, world.ErrInvalidWorldState) {
		t.Fatalf("AddPlayer(missing room) error = %v, want ErrInvalidWorldState", err)
	}
}

func TestWorldPlayerCountEventsTrackCommittedChanges(t *testing.T) {
	gameWorld := loadTestWorld(t)

	aliceEvent, err := gameWorld.AddPlayer("Alice", "harbor_square")
	if err != nil {
		t.Fatalf("AddPlayer(Alice) returned an error: %v", err)
	}
	if aliceEvent == nil || aliceEvent.Count != 1 || aliceEvent.Revision != 1 {
		t.Fatalf("Alice player-count event = %+v, want count 1 at revision 1", aliceEvent)
	}

	bobEvent, err := gameWorld.AddPlayer("Bob", "harbor_square")
	if err != nil {
		t.Fatalf("AddPlayer(Bob) returned an error: %v", err)
	}
	if bobEvent == nil || bobEvent.Count != 2 || bobEvent.Revision != 2 {
		t.Fatalf("Bob player-count event = %+v, want count 2 at revision 2", bobEvent)
	}

	_, _, _, removalEvent, err := gameWorld.RemovePlayer("Alice")
	if err != nil {
		t.Fatalf("RemovePlayer(Alice) returned an error: %v", err)
	}
	if removalEvent == nil || removalEvent.Count != 1 || removalEvent.Revision != 3 {
		t.Fatalf("removal player-count event = %+v, want count 1 at revision 3", removalEvent)
	}
}

func TestWorldResolvesNPCIDsAndNamesInPlayerRoom(t *testing.T) {
	gameWorld := loadTestWorld(t)
	if _, err := gameWorld.AddPlayer("Alice", "harbor_square"); err != nil {
		t.Fatalf("AddPlayer(Alice) returned an error: %v", err)
	}
	for _, identifier := range []string{"captain_eliza", "CAPTAIN ELIZA"} {
		if got, err := gameWorld.ResolveNPCID("Alice", identifier); err != nil || got != "captain_eliza" {
			t.Errorf("ResolveNPCID(%q) = (%q, %v), want captain_eliza", identifier, got, err)
		}
	}
	if _, err := gameWorld.ResolveNPCID("Alice", "Dock Warden"); !errors.Is(err, world.ErrNPCNotFound) {
		t.Errorf("ResolveNPCID(NPC in another room) error = %v, want ErrNPCNotFound", err)
	}
	if _, err := gameWorld.ResolveNPCID("Alice", " "); !errors.Is(err, world.ErrInvalidArgument) {
		t.Errorf("ResolveNPCID(empty identifier) error = %v, want ErrInvalidArgument", err)
	}
	if _, err := gameWorld.ResolveNPCID("Unknown", "Captain Eliza"); !errors.Is(err, world.ErrPlayerNotFound) {
		t.Errorf("ResolveNPCID(unknown player) error = %v, want ErrPlayerNotFound", err)
	}
}

func TestWorldDisconnectRemovesPlayerFromActiveCombat(t *testing.T) {
	gameWorld := loadTestWorld(t)
	for _, name := range []string{"Alice", "Bob"} {
		if _, err := gameWorld.AddPlayer(name, "docks"); err != nil {
			t.Fatalf("AddPlayer(%s) returned an error: %v", name, err)
		}
	}
	if _, err := gameWorld.CreateGroup("Alice", "Crew"); err != nil {
		t.Fatalf("CreateGroup() returned an error: %v", err)
	}
	if _, err := gameWorld.InvitePlayer("Alice", "Bob"); err != nil {
		t.Fatalf("InvitePlayer() returned an error: %v", err)
	}
	if _, _, err := gameWorld.JoinGroup("Bob", "Crew"); err != nil {
		t.Fatalf("JoinGroup() returned an error: %v", err)
	}
	combat, _, err := gameWorld.StartCombat("Alice", "Dock Warden")
	if err != nil {
		t.Fatalf("StartCombat() returned an error: %v", err)
	}
	if combat.TurnIndex < 0 || combat.Participants[combat.TurnIndex].ID != "Alice" {
		t.Fatalf("initial turn = %+v, want Alice to own the active turn", combat)
	}
	_, _, combatEvents, _, err := gameWorld.RemovePlayer("Alice")
	if err != nil {
		t.Fatalf("RemovePlayer(Alice) returned an error: %v", err)
	}
	if len(combatEvents) == 0 {
		t.Fatalf("combat progression events = %v, want removal progression", combatEvents)
	}
	if _, err := gameWorld.GetPlayerReconnectRoom("Alice"); !errors.Is(err, world.ErrPlayerNotInActiveCombat) {
		t.Fatalf("GetPlayerReconnectRoom(Alice) error = %v, want ErrPlayerNotInActiveCombat", err)
	}
	var bobTurn bool
	for _, event := range combatEvents {
		if turn, ok := event.(world.CombatTurnChanged); ok &&
			turn.IsPlayer && turn.ParticipantID == "Bob" {
			bobTurn = true
		}
	}
	if !bobTurn {
		t.Fatalf("combat events after Alice disconnect = %+v, want Bob turn", combatEvents)
	}

	if _, err := gameWorld.AddPlayer("Alice", "harbor_square"); err != nil {
		t.Fatalf("AddPlayer(Alice) after combat disconnect returned an error: %v", err)
	}
	if count, err := gameWorld.GetPlayerCount(); err != nil || count != 2 {
		t.Fatalf("player count after reconnect = (%d, %v), want Alice and Bob", count, err)
	}
	reconnected, err := gameWorld.GetCombat(combat.ID)
	if err != nil {
		t.Fatalf("GetCombat() after reconnect returned an error: %v", err)
	}
	for _, participant := range reconnected.Participants {
		if participant.IsPlayer && participant.ID == "Alice" {
			t.Fatal("Alice remains in combat after disconnecting")
		}
	}
}

func TestWorldCombatRoundQueriesAndTurnAdvancement(t *testing.T) {
	gameWorld := loadTestWorld(t)
	if _, err := gameWorld.AddPlayer("Alice", "docks"); err != nil {
		t.Fatalf("AddPlayer(Alice) returned an error: %v", err)
	}
	combat, _, err := gameWorld.StartCombat("Alice", "Dock Warden")
	if err != nil {
		t.Fatalf("StartCombat() returned an error: %v", err)
	}
	if combat.TurnIndex < 0 || combat.Participants[combat.TurnIndex].ID != "Alice" {
		t.Fatalf("initial turn = %+v, want Alice", combat)
	}

	if _, err := gameWorld.SkipPlayerCombatTurn(combat.ID, "Alice"); err != nil {
		t.Fatalf("SkipPlayerCombatTurn(Alice) returned an error: %v", err)
	}
	combat, err = gameWorld.GetCombat(combat.ID)
	if err != nil {
		t.Fatalf("GetCombat() after skipping Alice returned an error: %v", err)
	}
	if combat.TurnIndex < 0 || combat.TurnIndex >= len(combat.Participants) {
		t.Fatalf("unexpected combat turn after Alice skipped: %+v", combat)
	}
	currentTurn, err := gameWorld.GetCurrentCombatTurn(combat.ID)
	if err != nil {
		t.Fatalf("GetCurrentCombatTurn() after skipping Alice returned an error: %v", err)
	}
	if currentTurn == nil || currentTurn.ParticipantID != combat.Participants[combat.TurnIndex].ID {
		t.Fatalf("current turn = %+v, combat turn index = %d", currentTurn, combat.TurnIndex)
	}
}

func TestWorldSkipPlayerCombatTurnRequiresCurrentParticipant(t *testing.T) {
	gameWorld := loadTestWorld(t)
	if _, err := gameWorld.AddPlayer("Alice", "docks"); err != nil {
		t.Fatalf("AddPlayer(Alice) returned an error: %v", err)
	}
	combat, _, err := gameWorld.StartCombat("Alice", "Dock Warden")
	if err != nil {
		t.Fatalf("StartCombat() returned an error: %v", err)
	}
	turnEvent, err := gameWorld.GetCurrentCombatTurn(combat.ID)
	if err != nil {
		t.Fatalf("GetCurrentCombatTurn() returned an error: %v", err)
	}
	if turnEvent == nil || !turnEvent.IsPlayer {
		t.Fatalf("initial turn event = %+v, want Alice's turn", turnEvent)
	}
	if _, err := gameWorld.SkipPlayerCombatTurn(combat.ID, "dock_warden"); !errors.Is(err, world.ErrInvalidCombatTurn) {
		t.Fatalf("SkipPlayerCombatTurn(other participant) error = %v, want ErrInvalidCombatTurn", err)
	}

	nextEvents, err := gameWorld.SkipPlayerCombatTurn(combat.ID, "Alice")
	if err != nil {
		t.Fatalf("SkipPlayerCombatTurn(Alice) returned an error: %v", err)
	}
	var turnChanged bool
	for _, event := range nextEvents {
		if turn, ok := event.(world.CombatTurnChanged); ok {
			turnChanged = true
			if turn.ParticipantID == "" {
				t.Fatalf("next turn event has no participant: %+v", turn)
			}
			break
		}
	}
	if !turnChanged {
		t.Fatalf("next turn after skipping Alice = %+v, want a turn event", nextEvents)
	}
	combat, err = gameWorld.GetCombat(combat.ID)
	if err != nil {
		t.Fatalf("GetCombat() returned an error: %v", err)
	}
	if combat.Participants[0].SpeedPoints < 0 {
		t.Fatalf("Alice's speed points = %d, want a non-negative value", combat.Participants[0].SpeedPoints)
	}
}

func TestWorldUseItemReportsMissingAndNonConsumableItems(t *testing.T) {
	gameWorld := loadTestWorld(t)
	if _, err := gameWorld.AddPlayer("Alice", "harbor_square"); err != nil {
		t.Fatalf("AddPlayer(Alice) returned an error: %v", err)
	}
	if _, err := gameWorld.UseItem("Alice", "item.healing_potion"); !errors.Is(err, world.ErrItemNotInInventory) {
		t.Fatalf("UseItem(missing potion) error = %v, want ErrItemNotInInventory", err)
	}
	if err := gameWorld.TakeItem("Alice", "item.rope"); err != nil {
		t.Fatalf("TakeItem(rope) returned an error: %v", err)
	}
	if _, err := gameWorld.UseItem("Alice", "item.rope"); !errors.Is(err, world.ErrItemNotConsumable) {
		t.Fatalf("UseItem(rope) error = %v, want ErrItemNotConsumable", err)
	}
	if _, err := gameWorld.UseItem("Unknown", "item.rope"); !errors.Is(err, world.ErrPlayerNotFound) {
		t.Fatalf("UseItem(unknown player) error = %v, want ErrPlayerNotFound", err)
	}
}

func TestWorldUseItemHealsAfterCombatAndPreservesUnusedPotion(t *testing.T) {
	gameWorld := loadTestWorld(t)
	if _, err := gameWorld.AddPlayer("Alice", "apothecary"); err != nil {
		t.Fatalf("AddPlayer(Alice) returned an error: %v", err)
	}
	if err := gameWorld.TakeItem("Alice", "item.healing_potion"); err != nil {
		t.Fatalf("TakeItem(healing potion) returned an error: %v", err)
	}
	for _, direction := range []string{"west", "north", "south", "south", "south"} {
		if _, err := gameWorld.MovePlayer("Alice", direction); err != nil {
			t.Fatalf("MovePlayer(%s) returned an error: %v", direction, err)
		}
	}

	combat, _, err := gameWorld.StartCombat("Alice", "Dock Warden")
	if err != nil {
		t.Fatalf("StartCombat() returned an error: %v", err)
	}
	if _, err := gameWorld.UseItem("Alice", "item.healing_potion"); !errors.Is(err, world.ErrPlayerAlreadyInCombat) {
		t.Fatalf("UseItem(in combat) error = %v, want ErrPlayerAlreadyInCombat", err)
	}
	if _, _, _, _, err := gameWorld.FleeCombat(combat.ID, "Alice"); err != nil {
		t.Fatalf("FleeCombat() returned an error: %v", err)
	}

	before, err := gameWorld.GetPlayerCombatStats("Alice")
	if err != nil {
		t.Fatalf("GetPlayerCombatStats() returned an error: %v", err)
	}
	if before.HP >= before.MaxHP {
		t.Fatalf("HP before healing = %d, want less than max HP %d", before.HP, before.MaxHP)
	}
	used, err := gameWorld.UseItem("Alice", "item.healing_potion")
	if err != nil {
		t.Fatalf("UseItem(healing potion) returned an error: %v", err)
	}
	healed, err := gameWorld.GetPlayerCombatStats("Alice")
	if err != nil {
		t.Fatalf("GetPlayerCombatStats() after healing returned an error: %v", err)
	}
	expectedHealing := 40
	if expectedHealing > before.MaxHP-before.HP {
		expectedHealing = before.MaxHP - before.HP
	}
	if healed.HP != before.HP+expectedHealing ||
		used.StatChanges[world.CombatStatHP] != expectedHealing {
		t.Fatalf("healing result = (%d HP, %+v), want %d HP healed", healed.HP, used, expectedHealing)
	}
	for _, direction := range []string{"north", "north", "east"} {
		if _, err := gameWorld.MovePlayer("Alice", direction); err != nil {
			t.Fatalf("MovePlayer(%s) after healing returned an error: %v", direction, err)
		}
	}
	if err := gameWorld.TakeItem("Alice", "item.healing_potion"); err != nil {
		t.Fatalf("TakeItem(second healing potion) returned an error: %v", err)
	}
	if _, err := gameWorld.UseItem("Alice", "item.healing_potion"); !errors.Is(err, world.ErrItemHasNoEffect) {
		t.Fatalf("UseItem(at max HP) error = %v, want ErrItemHasNoEffect", err)
	}
	inventory, err := gameWorld.GetInventoryWithQuantities("Alice")
	if err != nil {
		t.Fatalf("GetInventoryWithQuantities() returned an error: %v", err)
	}
	if len(inventory) != 1 || inventory[0].ID != "item.healing_potion" || inventory[0].Quantity != 1 {
		t.Fatalf("inventory after healing = %+v, want one unused potion", inventory)
	}
}

func TestWorldCombatRejectsInvalidAttackAndEffectRequests(t *testing.T) {
	gameWorld := loadTestWorld(t)
	if _, err := gameWorld.AddPlayer("Alice", "docks"); err != nil {
		t.Fatalf("AddPlayer(Alice) returned an error: %v", err)
	}
	combat, _, err := gameWorld.StartCombat("Alice", "Dock Warden")
	if err != nil {
		t.Fatalf("StartCombat() returned an error: %v", err)
	}
	if _, _, _, err := gameWorld.AttackCombatParticipant(combat.ID, "Alice", "dock_warden"); err != nil {
		t.Fatalf("AttackCombatParticipant(valid initialized attack) returned an error: %v", err)
	}
	if _, _, _, err := gameWorld.AttackCombatParticipant(combat.ID, "dock_warden", "Alice"); !errors.Is(err, world.ErrInvalidCombatTurn) {
		t.Fatalf("AttackCombatParticipant(out of turn) error = %v, want ErrInvalidCombatTurn", err)
	}
	if _, _, _, err := gameWorld.AttackCombatParticipant(combat.ID, "Alice", "Alice"); !errors.Is(err, world.ErrInvalidCombatTarget) {
		t.Fatalf("AttackCombatParticipant(self target) error = %v, want ErrInvalidCombatTarget", err)
	}
	if _, _, _, err := gameWorld.AttackCombatParticipant(combat.ID, "Alice", "missing"); !errors.Is(err, world.ErrInvalidCombatTarget) {
		t.Fatalf("AttackCombatParticipant(unknown target) error = %v, want ErrInvalidCombatTarget", err)
	}

	effect := world.CombatEffect{
		ID:             "test-effect",
		Name:           "Test Effect",
		TargetID:       "Alice",
		Stat:           world.CombatStatDefense,
		Modifier:       2,
		Duration:       2,
		RemainingTurns: 2,
	}
	invalidEffects := []struct {
		name   string
		effect world.CombatEffect
	}{
		{name: "empty ID", effect: world.CombatEffect{Name: "Effect", TargetID: "Alice", Stat: world.CombatStatDefense, Modifier: 1, Duration: 1, RemainingTurns: 1}},
		{name: "wrong target", effect: world.CombatEffect{ID: "wrong-target", Name: "Effect", TargetID: "dock_warden", Stat: world.CombatStatDefense, Modifier: 1, Duration: 1, RemainingTurns: 1}},
		{name: "invalid duration", effect: world.CombatEffect{ID: "duration", Name: "Effect", TargetID: "Alice", Stat: world.CombatStatDefense, Modifier: 1, Duration: 1, RemainingTurns: 2}},
		{name: "no modifier", effect: world.CombatEffect{ID: "no-modifier", Name: "Effect", TargetID: "Alice", Stat: world.CombatStatDefense, Duration: 1, RemainingTurns: 1}},
	}
	for _, test := range invalidEffects {
		t.Run(test.name, func(t *testing.T) {
			if _, err := gameWorld.ApplyCombatEffect(combat.ID, "Alice", test.effect); !errors.Is(err, world.ErrInvalidArgument) {
				t.Fatalf("ApplyCombatEffect() error = %v, want ErrInvalidArgument", err)
			}
		})
	}
	if _, err := gameWorld.ApplyCombatEffect(combat.ID, "Alice", effect); err != nil {
		t.Fatalf("ApplyCombatEffect(valid) returned an error: %v", err)
	}
	if _, err := gameWorld.ApplyCombatEffect(combat.ID, "Alice", effect); !errors.Is(err, world.ErrInvalidWorldState) {
		t.Fatalf("ApplyCombatEffect(duplicate) error = %v, want ErrInvalidWorldState", err)
	}
	if _, err := gameWorld.RemoveCombatEffect(combat.ID, "Alice", "missing"); !errors.Is(err, world.ErrInvalidWorldState) {
		t.Fatalf("RemoveCombatEffect(missing effect) error = %v, want ErrInvalidWorldState", err)
	}
	if _, err := gameWorld.RemoveCombatEffect(combat.ID, "missing", effect.ID); !errors.Is(err, world.ErrInvalidWorldState) {
		t.Fatalf("RemoveCombatEffect(missing participant) error = %v, want ErrInvalidWorldState", err)
	}
}
