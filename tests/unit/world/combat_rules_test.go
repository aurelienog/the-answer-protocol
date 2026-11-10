package internal_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"the-answer-protocol/internal/world"
)

func TestWorldCombatVictoryRewardsEveryConnectedParticipant(t *testing.T) {
	gameWorld := loadTestWorld(t)
	for _, name := range []string{"Alice", "Bob"} {
		if _, err := gameWorld.AddPlayer(name, "forest_edge"); err != nil {
			t.Fatalf("AddPlayer(%s) returned an error: %v", name, err)
		}
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

	combat, started, err := gameWorld.StartCombat("Alice", "Dire Wolf")
	if err != nil {
		t.Fatalf("StartCombat() returned an error: %v", err)
	}
	if !reflect.DeepEqual(started.Participants, []string{"Alice", "Bob"}) {
		t.Fatalf("combat participants = %v, want Alice and Bob", started.Participants)
	}
	combat, _, err = gameWorld.InitializeCombatTurns(combat.ID)
	if err != nil {
		t.Fatalf("InitializeCombatTurns() returned an error: %v", err)
	}

	combatEnded := false
	for actions := 0; actions < 100; actions++ {
		combat, err = gameWorld.GetCombat(combat.ID)
		if err != nil {
			t.Fatalf("GetCombat() returned an error: %v", err)
		}
		if !combat.Active {
			break
		}
		npcAlive := false
		for _, candidate := range combat.Participants {
			if !candidate.IsPlayer && candidate.Alive {
				npcAlive = true
				break
			}
		}
		if !npcAlive {
			if _, err := gameWorld.EndCombat(combat.ID); err != nil {
				t.Fatalf("EndCombat() after NPC defeat returned an error: %v", err)
			}
			combatEnded = true
			break
		}
		if combat.TurnIndex < 0 {
			combat, _, err = gameWorld.StartCombatRound(combat.ID)
			if err != nil {
				t.Fatalf("StartCombatRound() returned an error: %v", err)
			}
			continue
		}

		participant := combat.Participants[combat.TurnIndex]
		if participant.IsPlayer {
			_, _, _, _, _, err = gameWorld.AttackCombatParticipant(
				combat.ID,
				participant.ID,
				"dire_wolf",
			)
		} else {
			_, _, _, _, _, err = gameWorld.ExecuteNPCCombatTurn(combat.ID)
		}
		if err != nil {
			t.Fatalf("combat action for %q returned an error: %v", participant.ID, err)
		}
		if actions == 99 {
			t.Fatal("combat did not resolve within 100 actions")
		}
	}

	combat, err = gameWorld.GetCombat(combat.ID)
	if err != nil {
		t.Fatalf("GetCombat() before ending returned an error: %v", err)
	}
	if combat.Active {
		t.Fatal("combat remains active after the Dire Wolf was defeated")
	}
	if !combatEnded {
		combat, err = gameWorld.EndCombat(combat.ID)
		if err != nil {
			t.Fatalf("EndCombat() returned an error: %v", err)
		}
	}
	if combat.Active {
		t.Fatal("combat is active after EndCombat()")
	}
	for _, name := range []string{"Alice", "Bob"} {
		inventory, err := gameWorld.GetInventoryWithQuantities(name)
		if err != nil {
			t.Fatalf("GetInventoryWithQuantities(%s) returned an error: %v", name, err)
		}
		foundReward := false
		for _, item := range inventory {
			if item.ID == "item.herbs" && item.Quantity == 1 {
				foundReward = true
			}
		}
		if !foundReward {
			t.Errorf("%s's inventory = %+v, want one Dire Wolf reward", name, inventory)
		}
	}
	if _, err := gameWorld.GetActiveCombatForPlayer("Alice"); !errors.Is(err, world.ErrPlayerNotInActiveCombat) {
		t.Fatalf("GetActiveCombatForPlayer(Alice) after victory error = %v", err)
	}
}

func TestWorldCombatHealingItemRestoresHealthWithoutSpendingTurn(t *testing.T) {
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
	combat, _, err = gameWorld.InitializeCombatTurns(combat.ID)
	if err != nil {
		t.Fatalf("InitializeCombatTurns() returned an error: %v", err)
	}
	if combat.Participants[combat.TurnIndex].ID != "Alice" {
		t.Fatalf("initial combat turn = %q, want Alice", combat.Participants[combat.TurnIndex].ID)
	}
	if _, _, _, _, _, err := gameWorld.AttackCombatParticipant(combat.ID, "Alice", "dock_warden"); err != nil {
		t.Fatalf("Alice's first attack returned an error: %v", err)
	}
	combat, err = gameWorld.GetCombat(combat.ID)
	if err != nil {
		t.Fatalf("GetCombat() after first attack returned an error: %v", err)
	}
	if combat.TurnIndex < 0 || combat.Participants[combat.TurnIndex].IsPlayer {
		t.Fatalf("turn after Alice's attack = %+v, want NPC turn", combat)
	}
	if _, _, _, _, _, err := gameWorld.ExecuteNPCCombatTurn(combat.ID); err != nil {
		t.Fatalf("NPC turn returned an error: %v", err)
	}
	combat, err = gameWorld.GetCombat(combat.ID)
	if err != nil {
		t.Fatalf("GetCombat() after NPC turn returned an error: %v", err)
	}
	if combat.TurnIndex < 0 {
		combat, _, err = gameWorld.StartCombatRound(combat.ID)
		if err != nil {
			t.Fatalf("StartCombatRound() returned an error: %v", err)
		}
	}
	if combat.Participants[combat.TurnIndex].ID != "Alice" {
		t.Fatalf("combat turn before potion use = %q, want Alice", combat.Participants[combat.TurnIndex].ID)
	}
	hpBefore, err := gameWorld.GetPlayerCombatStats("Alice")
	if err != nil {
		t.Fatalf("GetPlayerCombatStats() returned an error: %v", err)
	}
	actionsBefore := world.AvailableActions(combat.Participants[combat.TurnIndex])

	combat, used, err := gameWorld.CombatUseItem(combat.ID, "Alice", "item.healing_potion")
	if err != nil {
		t.Fatalf("CombatUseItem() returned an error: %v", err)
	}
	expectedHealing := hpBefore.MaxHP - hpBefore.HP
	if expectedHealing > 40 {
		expectedHealing = 40
	}
	if used.StatChanges[world.CombatStatHP] != expectedHealing {
		t.Fatalf("healing change = %d, want %d", used.StatChanges[world.CombatStatHP], expectedHealing)
	}
	if combat.Participants[combat.TurnIndex].ID != "Alice" ||
		world.AvailableActions(combat.Participants[combat.TurnIndex]) != actionsBefore {
		t.Fatalf("turn/actions after potion use = (%d, %d), want Alice with %d actions",
			combat.TurnIndex, world.AvailableActions(combat.Participants[combat.TurnIndex]), actionsBefore)
	}
	hpAfter, err := gameWorld.GetPlayerCombatStats("Alice")
	if err != nil {
		t.Fatalf("GetPlayerCombatStats() after potion returned an error: %v", err)
	}
	if hpAfter.HP != hpBefore.HP+expectedHealing {
		t.Fatalf("HP after potion = %d, want %d", hpAfter.HP, hpBefore.HP+expectedHealing)
	}
	inventory, err := gameWorld.GetInventory("Alice")
	if err != nil {
		t.Fatalf("GetInventory() after potion returned an error: %v", err)
	}
	for _, item := range inventory {
		if item.ID == "item.healing_potion" {
			t.Fatal("healing potion remains in inventory after use")
		}
	}
}

func TestWorldFleeCombatAppliesNonlethalPenaltyAndEndsEncounter(t *testing.T) {
	gameWorld := loadTestWorld(t)
	if _, err := gameWorld.AddPlayer("Alice", "docks"); err != nil {
		t.Fatalf("AddPlayer(Alice) returned an error: %v", err)
	}
	combat, _, err := gameWorld.StartCombat("Alice", "Dock Warden")
	if err != nil {
		t.Fatalf("StartCombat() returned an error: %v", err)
	}
	combat, _, err = gameWorld.InitializeCombatTurns(combat.ID)
	if err != nil {
		t.Fatalf("InitializeCombatTurns() returned an error: %v", err)
	}

	combat, event, err := gameWorld.FleeCombat(combat.ID, "Alice")
	if err != nil {
		t.Fatalf("FleeCombat() returned an error: %v", err)
	}
	if event == nil || event.Damage != 15 || event.RemainingHP != 85 {
		t.Fatalf("flee event = %+v, want 15 damage and 85 HP", event)
	}
	if !combat.Participants[0].Escaped {
		t.Fatal("Alice is not marked as escaped in combat")
	}
	combat, err = gameWorld.EndCombat(combat.ID)
	if err != nil {
		t.Fatalf("EndCombat() after flee returned an error: %v", err)
	}
	if combat.Active {
		t.Fatal("combat remains active after the only player flees")
	}
	stats, err := gameWorld.GetPlayerCombatStats("Alice")
	if err != nil {
		t.Fatalf("GetPlayerCombatStats() returned an error: %v", err)
	}
	if stats.HP != 85 {
		t.Fatalf("Alice's HP after fleeing = %d, want 85", stats.HP)
	}
	if roomID, err := gameWorld.GetPlayerRoom("Alice"); err != nil || roomID != "docks" {
		t.Fatalf("Alice's room after fleeing = (%q, %v), want docks", roomID, err)
	}
}

func TestWorldChatScopesSortRecipientsAndRequireGroupMembership(t *testing.T) {
	gameWorld := loadTestWorld(t)
	for name, roomID := range map[string]string{
		"Alice": "harbor_square",
		"Bob":   "harbor_square",
		"Carol": "market_street",
	} {
		if _, err := gameWorld.AddPlayer(name, roomID); err != nil {
			t.Fatalf("AddPlayer(%s) returned an error: %v", name, err)
		}
	}
	if _, err := gameWorld.CreateGroup("Alice", "Harbor Crew"); err != nil {
		t.Fatalf("CreateGroup() returned an error: %v", err)
	}
	if _, err := gameWorld.InvitePlayer("Alice", "Bob"); err != nil {
		t.Fatalf("InvitePlayer() returned an error: %v", err)
	}
	if _, _, err := gameWorld.JoinGroup("Bob", "Harbor Crew"); err != nil {
		t.Fatalf("JoinGroup() returned an error: %v", err)
	}

	roomChat, err := gameWorld.SendChat("Alice", " room ", "  hello harbor  ")
	if err != nil {
		t.Fatalf("SendChat(room) returned an error: %v", err)
	}
	if roomChat.Scope != "ROOM" || roomChat.Message != "hello harbor" ||
		!reflect.DeepEqual(roomChat.Recipients, []string{"Alice", "Bob"}) {
		t.Fatalf("room chat = %+v", roomChat)
	}
	globalChat, err := gameWorld.SendChat("Alice", "global", "hello world")
	if err != nil {
		t.Fatalf("SendChat(global) returned an error: %v", err)
	}
	if !reflect.DeepEqual(globalChat.Recipients, []string{"Alice", "Bob", "Carol"}) {
		t.Fatalf("global chat recipients = %v, want sorted players", globalChat.Recipients)
	}
	groupChat, err := gameWorld.SendChat("Alice", "group", "hello crew")
	if err != nil {
		t.Fatalf("SendChat(group) returned an error: %v", err)
	}
	if !reflect.DeepEqual(groupChat.Recipients, []string{"Alice", "Bob"}) {
		t.Fatalf("group chat recipients = %v, want Alice and Bob", groupChat.Recipients)
	}
	groupChat.Recipients[0] = "Carol"
	groupChat, err = gameWorld.SendChat("Alice", "GROUP", "another message")
	if err != nil {
		t.Fatalf("SendChat(group) after recipient mutation returned an error: %v", err)
	}
	if !reflect.DeepEqual(groupChat.Recipients, []string{"Alice", "Bob"}) {
		t.Fatalf("mutating chat recipients changed group membership: %v", groupChat.Recipients)
	}
	if _, err := gameWorld.SendChat("Carol", "GROUP", "not a member"); !errors.Is(err, world.ErrPlayerNotInGroup) {
		t.Fatalf("SendChat(group) for nonmember error = %v, want ErrPlayerNotInGroup", err)
	}
	if _, err := gameWorld.SendChat("Alice", "LOCAL", "unsupported"); !errors.Is(err, world.ErrInvalidArgument) {
		t.Fatalf("SendChat(unsupported scope) error = %v, want ErrInvalidArgument", err)
	}
}

func TestWorldCombatHelpAndJoinRequireRoomAndHelpRequest(t *testing.T) {
	gameWorld := loadTestWorld(t)
	for name, roomID := range map[string]string{
		"Alice": "forest_edge",
		"Bob":   "forest_edge",
		"Carol": "harbor_square",
	} {
		if _, err := gameWorld.AddPlayer(name, roomID); err != nil {
			t.Fatalf("AddPlayer(%s) returned an error: %v", name, err)
		}
	}
	combat, _, err := gameWorld.StartCombat("Alice", "Dire Wolf")
	if err != nil {
		t.Fatalf("StartCombat() returned an error: %v", err)
	}
	if _, _, err := gameWorld.JoinCombat("Bob", combat.ID); !errors.Is(err, world.ErrCombatHelpNotRequested) {
		t.Fatalf("JoinCombat() before help request error = %v, want ErrCombatHelpNotRequested", err)
	}

	helpCombat, helpEvent, err := gameWorld.RequestCombatHelp("Alice")
	if err != nil {
		t.Fatalf("RequestCombatHelp() returned an error: %v", err)
	}
	if !helpCombat.HelpRequested || helpEvent.PlayerName != "Alice" ||
		helpEvent.RoomID != "forest_edge" ||
		!reflect.DeepEqual(helpEvent.Recipients, []string{"Alice", "Bob"}) {
		t.Fatalf("combat help result = (%+v, %+v)", helpCombat, helpEvent)
	}
	if _, _, err := gameWorld.JoinCombat("Carol", combat.ID); !errors.Is(err, world.ErrInvalidWorldState) {
		t.Fatalf("JoinCombat() from another room error = %v, want ErrInvalidWorldState", err)
	}

	joinedCombat, joinedEvent, err := gameWorld.JoinCombat("Bob", combat.ID)
	if err != nil {
		t.Fatalf("JoinCombat(Bob) returned an error: %v", err)
	}
	if joinedEvent.PlayerName != "Bob" ||
		!reflect.DeepEqual(joinedEvent.Recipients, []string{"Alice", "Bob"}) {
		t.Fatalf("combat join event = %+v", joinedEvent)
	}
	bobJoined := false
	for _, participant := range joinedCombat.Participants {
		if participant.IsPlayer && participant.ID == "Bob" {
			bobJoined = true
			break
		}
	}
	if len(joinedCombat.Participants) != 3 || !bobJoined {
		t.Fatalf("participants after Bob joins = %+v", joinedCombat.Participants)
	}
	if _, _, err := gameWorld.JoinCombat("Bob", combat.ID); !errors.Is(err, world.ErrPlayerAlreadyInCombat) {
		t.Fatalf("repeated JoinCombat(Bob) error = %v, want ErrPlayerAlreadyInCombat", err)
	}
}

func TestWorldRemovePlayerReturnsUniqueItemToRoom(t *testing.T) {
	gameWorld := loadWorldWithUniqueItemInHarbor(t)
	for _, name := range []string{"Alice", "Bob"} {
		if _, err := gameWorld.AddPlayer(name, "harbor_square"); err != nil {
			t.Fatalf("AddPlayer(%s) returned an error: %v", name, err)
		}
	}
	if err := gameWorld.TakeItem("Alice", "item.gold_coins"); err != nil {
		t.Fatalf("TakeItem(gold coins) returned an error: %v", err)
	}

	left, returned, _, combatsToEnd, _, err := gameWorld.RemovePlayer("Alice")
	if err != nil {
		t.Fatalf("RemovePlayer(Alice) returned an error: %v", err)
	}
	if left.PlayerName != "Alice" || left.RoomID != "harbor_square" ||
		!reflect.DeepEqual(left.RoomPlayers, []string{"Alice", "Bob"}) {
		t.Fatalf("player-left event = %+v", left)
	}
	if returned.PlayerName != "Alice" || returned.RoomID != "harbor_square" ||
		len(returned.ItemIDs) != 1 || returned.ItemIDs[0] == "item.gold_coins" {
		t.Fatalf("unique-items-returned event = %+v, want one concrete instance ID", returned)
	}
	if len(combatsToEnd) != 0 {
		t.Fatalf("combats to end without active combat = %v, want none", combatsToEnd)
	}
	if _, err := gameWorld.GetPlayerRoom("Alice"); !errors.Is(err, world.ErrPlayerNotFound) {
		t.Fatalf("GetPlayerRoom(Alice) after removal error = %v, want ErrPlayerNotFound", err)
	}
	room, err := gameWorld.GetRoom("harbor_square")
	if err != nil {
		t.Fatalf("GetRoom(harbor_square) returned an error: %v", err)
	}
	if room.Items["item.gold_coins"] != 1 {
		t.Fatalf("gold coins in harbor after removal = %d, want 1", room.Items["item.gold_coins"])
	}
}

func TestWorldUniqueItemAddsCombatStats(t *testing.T) {
	gameWorld := loadWorldWithUniqueItemInHarbor(t)
	if _, err := gameWorld.AddPlayer("Alice", "harbor_square"); err != nil {
		t.Fatalf("AddPlayer(Alice) returned an error: %v", err)
	}
	if err := gameWorld.TakeItem("Alice", "item.gold_coins"); err != nil {
		t.Fatalf("TakeItem(gold coins) returned an error: %v", err)
	}
	for _, direction := range []string{"north", "north", "north"} {
		if _, err := gameWorld.MovePlayer("Alice", direction); err != nil {
			t.Fatalf("MovePlayer(%s) returned an error: %v", direction, err)
		}
	}

	combat, _, err := gameWorld.StartCombat("Alice", "Dire Wolf")
	if err != nil {
		t.Fatalf("StartCombat() returned an error: %v", err)
	}

	for _, participant := range combat.Participants {
		if participant.IsPlayer && participant.ID == "Alice" {
			if participant.Stats.Attack != world.DefaultPlayerAttack+7 {
				t.Fatalf("Alice's attack in combat = %d, want %d", participant.Stats.Attack, world.DefaultPlayerAttack+7)
			}
			return
		}
	}
	t.Fatal("Alice is missing from combat participants")
}

func loadWorldWithUniqueItemInHarbor(t *testing.T) *world.World {
	t.Helper()

	data, err := os.ReadFile("../../../data/world.json")
	if err != nil {
		t.Fatalf("read world fixture: %v", err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode world fixture: %v", err)
	}
	var definitions map[string]world.Item
	if err := json.Unmarshal(document["items"], &definitions); err != nil {
		t.Fatalf("decode item definitions: %v", err)
	}
	uniqueItem := definitions["item.gold_coins"]
	uniqueItem.Attack = 7
	definitions["item.gold_coins"] = uniqueItem
	document["items"], err = json.Marshal(definitions)
	if err != nil {
		t.Fatalf("encode item definitions: %v", err)
	}
	var rooms map[string]json.RawMessage
	if err := json.Unmarshal(document["rooms"], &rooms); err != nil {
		t.Fatalf("decode rooms: %v", err)
	}
	var harbor map[string]json.RawMessage
	if err := json.Unmarshal(rooms["harbor_square"], &harbor); err != nil {
		t.Fatalf("decode harbor square: %v", err)
	}
	var items map[string]int
	if err := json.Unmarshal(harbor["items"], &items); err != nil {
		t.Fatalf("decode harbor items: %v", err)
	}
	items["item.gold_coins"] = 1
	harbor["items"], err = json.Marshal(items)
	if err != nil {
		t.Fatalf("encode harbor items: %v", err)
	}
	rooms["harbor_square"], err = json.Marshal(harbor)
	if err != nil {
		t.Fatalf("encode harbor square: %v", err)
	}
	document["rooms"], err = json.Marshal(rooms)
	if err != nil {
		t.Fatalf("encode rooms: %v", err)
	}
	data, err = json.Marshal(document)
	if err != nil {
		t.Fatalf("encode world fixture: %v", err)
	}

	path := filepath.Join(t.TempDir(), "world.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write world fixture: %v", err)
	}
	gameWorld, err := world.Load(path)
	if err != nil {
		t.Fatalf("load world fixture: %v", err)
	}
	return gameWorld
}
