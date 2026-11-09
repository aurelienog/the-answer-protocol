package internal_test

import (
	"errors"
	"testing"

	"the-answer-protocol/internal/world"
)

func TestWorldInventoryTakeDropAndQuantities(t *testing.T) {
	gameWorld := loadTestWorld(t)
	if err := gameWorld.AddPlayer("Alice", "harbor_square"); err != nil {
		t.Fatalf("AddPlayer(Alice) returned an error: %v", err)
	}
	if err := gameWorld.TakeItem("Alice", "item.rope"); err != nil {
		t.Fatalf("TakeItem(rope) returned an error: %v", err)
	}
	if _, err := gameWorld.MovePlayer("Alice", "north"); err != nil {
		t.Fatalf("MovePlayer(market street) returned an error: %v", err)
	}
	if err := gameWorld.TakeItem("Alice", "item.pouch"); err != nil {
		t.Fatalf("TakeItem(pouch) returned an error: %v", err)
	}
	inventory, err := gameWorld.GetInventoryWithQuantities("Alice")
	if err != nil {
		t.Fatalf("GetInventoryWithQuantities() returned an error: %v", err)
	}
	if len(inventory) != 2 {
		t.Fatalf("inventory quantities = %+v, want rope and pouch", inventory)
	}
	quantities := make(map[string]int)
	for _, item := range inventory {
		quantities[item.ID] = item.Quantity
	}
	if quantities["item.rope"] != 1 || quantities["item.pouch"] != 1 {
		t.Fatalf("inventory quantities = %+v, want one rope and one pouch", inventory)
	}

	event, err := gameWorld.DropItem("Alice", "item.pouch")
	if err != nil {
		t.Fatalf("DropItem(pouch) returned an error: %v", err)
	}
	if event.PlayerName != "Alice" || event.RoomID != "market_street" ||
		len(event.RoomPlayers) != 1 || event.RoomPlayers[0] != "Alice" {
		t.Fatalf("drop event = %+v", event)
	}
	inventory, err = gameWorld.GetInventoryWithQuantities("Alice")
	if err != nil {
		t.Fatalf("GetInventoryWithQuantities() after drop returned an error: %v", err)
	}
	if len(inventory) != 1 || inventory[0].ID != "item.rope" || inventory[0].Quantity != 1 {
		t.Fatalf("inventory after dropping pouch = %+v, want one rope", inventory)
	}
	room, err := gameWorld.GetRoom("market_street")
	if err != nil {
		t.Fatalf("GetRoom(market_street) returned an error: %v", err)
	}
	if room.Items["item.pouch"] != 2 {
		t.Fatalf("pouches in room after drop = %d, want 2", room.Items["item.pouch"])
	}

	if _, err := gameWorld.DropItem("Alice", "item.leather_boots"); !errors.Is(err, world.ErrItemNotInInventory) {
		t.Fatalf("DropItem(missing boots) error = %v, want ErrItemNotInInventory", err)
	}
}

func TestWorldGroupInvitationJoinAndLeaveLifecycle(t *testing.T) {
	gameWorld := loadTestWorld(t)
	for _, name := range []string{"Alice", "Bob"} {
		if err := gameWorld.AddPlayer(name, "harbor_square"); err != nil {
			t.Fatalf("AddPlayer(%s) returned an error: %v", name, err)
		}
	}

	groupID, err := gameWorld.CreateGroup("Alice", "Adventurers")
	if err != nil {
		t.Fatalf("CreateGroup() returned an error: %v", err)
	}
	invitation, err := gameWorld.InvitePlayer("Alice", "Bob")
	if err != nil {
		t.Fatalf("InvitePlayer() returned an error: %v", err)
	}
	if invitation.GroupID != groupID || invitation.Inviter != "Alice" || invitation.Invitee != "Bob" {
		t.Fatalf("invitation = %+v", invitation)
	}
	invitations, err := gameWorld.GetGroupInvitations("Bob")
	if err != nil {
		t.Fatalf("GetGroupInvitations() returned an error: %v", err)
	}
	if len(invitations) != 1 || invitations[0].ID != groupID ||
		invitations[0].Name != "Adventurers" {
		t.Fatalf("Bob's invitations = %+v, want invitation to %s", invitations, groupID)
	}

	joinedID, joined, err := gameWorld.JoinGroup("Bob", "Adventurers")
	if err != nil {
		t.Fatalf("JoinGroup() returned an error: %v", err)
	}
	if joinedID != groupID || joined.PlayerName != "Bob" || len(joined.Members) != 1 || joined.Members[0] != "Alice" {
		t.Fatalf("join result = (%q, %+v)", joinedID, joined)
	}
	invitations, err = gameWorld.GetGroupInvitations("Bob")
	if err != nil {
		t.Fatalf("GetGroupInvitations() after joining returned an error: %v", err)
	}
	if len(invitations) != 0 {
		t.Fatalf("Bob's invitations after joining = %+v, want none", invitations)
	}

	left, err := gameWorld.LeaveGroup("Bob")
	if err != nil {
		t.Fatalf("LeaveGroup() returned an error: %v", err)
	}
	if left.PlayerName != "Bob" || len(left.Members) != 1 || left.Members[0] != "Alice" {
		t.Fatalf("leave result = %+v", left)
	}
	group, err := gameWorld.GetPlayerGroup("Alice")
	if err != nil || len(group.Members) != 1 || group.Members[0] != "Alice" {
		t.Fatalf("Alice's group after Bob leaves = (%+v, %v)", group, err)
	}
	if _, err := gameWorld.GetPlayerGroup("Bob"); !errors.Is(err, world.ErrPlayerNotInGroup) {
		t.Fatalf("GetPlayerGroup(Bob) after leaving error = %v, want ErrPlayerNotInGroup", err)
	}
}

func TestWorldRemovePlayerClearsAllLivePlayerState(t *testing.T) {
	gameWorld := loadWorldWithUniqueItemInHarbor(t)
	for _, name := range []string{"Alice", "Bob"} {
		if err := gameWorld.AddPlayer(name, "harbor_square"); err != nil {
			t.Fatalf("AddPlayer(%s) returned an error: %v", name, err)
		}
	}
	if err := gameWorld.TakeItem("Alice", "item.rope"); err != nil {
		t.Fatalf("TakeItem(rope) returned an error: %v", err)
	}
	if err := gameWorld.TakeItem("Alice", "item.gold_coins"); err != nil {
		t.Fatalf("TakeItem(gold coins) returned an error: %v", err)
	}
	if _, err := gameWorld.MovePlayer("Alice", "north"); err != nil {
		t.Fatalf("MovePlayer(market street) returned an error: %v", err)
	}
	if _, err := gameWorld.TalkToNPC("Alice", "Quest Advisor"); err != nil {
		t.Fatalf("TalkToNPC(Quest Advisor) returned an error: %v", err)
	}
	if err := gameWorld.AcceptQuest("Alice", "quest.explore_arvendale"); err != nil {
		t.Fatalf("AcceptQuest() returned an error: %v", err)
	}
	if _, err := gameWorld.CreateGroup("Bob", "Adventurers"); err != nil {
		t.Fatalf("CreateGroup() returned an error: %v", err)
	}
	if _, err := gameWorld.InvitePlayer("Bob", "Alice"); err != nil {
		t.Fatalf("InvitePlayer() returned an error: %v", err)
	}
	if _, err := gameWorld.MovePlayer("Alice", "south"); err != nil {
		t.Fatalf("MovePlayer(harbor square) returned an error: %v", err)
	}
	if _, err := gameWorld.MovePlayer("Alice", "south"); err != nil {
		t.Fatalf("MovePlayer(docks) returned an error: %v", err)
	}
	combat, _, err := gameWorld.StartCombat("Alice", "Dock Warden")
	if err != nil {
		t.Fatalf("StartCombat() returned an error: %v", err)
	}
	if _, _, err := gameWorld.InitializeCombatTurns(combat.ID); err != nil {
		t.Fatalf("InitializeCombatTurns() returned an error: %v", err)
	}

	_, returned, _, combatsToEnd, err := gameWorld.RemovePlayer("Alice")
	if err != nil {
		t.Fatalf("RemovePlayer(Alice) returned an error: %v", err)
	}
	if len(returned.ItemIDs) != 1 || returned.ItemIDs[0] == "item.gold_coins" {
		t.Fatalf("unique items returned = %v, want one concrete gold-coins instance", returned.ItemIDs)
	}
	if len(combatsToEnd) != 1 || combatsToEnd[0] != combat.ID {
		t.Fatalf("combats to end = %v, want only %q", combatsToEnd, combat.ID)
	}
	if _, err := gameWorld.GetPlayerRoom("Alice"); !errors.Is(err, world.ErrPlayerNotFound) {
		t.Fatalf("GetPlayerRoom(Alice) after removal error = %v, want ErrPlayerNotFound", err)
	}
	combat, err = gameWorld.GetCombat(combat.ID)
	if err != nil {
		t.Fatalf("GetCombat() after disconnect returned an error: %v", err)
	}
	for _, participant := range combat.Participants {
		if participant.IsPlayer && participant.ID == "Alice" {
			t.Fatal("Alice remains in combat after disconnecting")
		}
	}
	if _, err := gameWorld.EndCombat(combat.ID); err != nil {
		t.Fatalf("EndCombat() after disconnect returned an error: %v", err)
	}

	if err := gameWorld.AddPlayer("Alice", "market_street"); err != nil {
		t.Fatalf("AddPlayer(Alice) after removal returned an error: %v", err)
	}
	quests, err := gameWorld.GetPlayerQuests("Alice")
	if err != nil || len(quests) != 0 {
		t.Fatalf("quests after reconnect = (%+v, %v), want none", quests, err)
	}
	inventory, err := gameWorld.GetInventory("Alice")
	if err != nil || len(inventory) != 0 {
		t.Fatalf("inventory after reconnect = (%+v, %v), want empty", inventory, err)
	}
	invitations, err := gameWorld.GetGroupInvitations("Alice")
	if err != nil || len(invitations) != 0 {
		t.Fatalf("invitations after reconnect = (%+v, %v), want none", invitations, err)
	}
	if _, err := gameWorld.GetPlayerGroup("Alice"); !errors.Is(err, world.ErrPlayerNotInGroup) {
		t.Fatalf("GetPlayerGroup(Alice) after reconnect error = %v, want ErrPlayerNotInGroup", err)
	}
	if room, err := gameWorld.GetRoom("docks"); err != nil || room.Items["item.gold_coins"] != 1 {
		t.Fatalf("gold coins in docks after disconnect = (%d, %v), want 1", room.Items["item.gold_coins"], err)
	}
	if room, err := gameWorld.GetRoom("harbor_square"); err != nil || room.Items["item.rope"] != 0 {
		t.Fatalf("rope in harbor after disconnect = (%d, %v), want 0", room.Items["item.rope"], err)
	}
}

func TestWorldQuestDiscoveryProgressAndCompletionQueries(t *testing.T) {
	gameWorld := loadTestWorld(t)
	if err := gameWorld.AddPlayer("Alice", "market_street"); err != nil {
		t.Fatalf("AddPlayer(Alice) returned an error: %v", err)
	}

	if _, err := gameWorld.TalkToNPC("Alice", "Quest Advisor"); err != nil {
		t.Fatalf("TalkToNPC(Quest Advisor) returned an error: %v", err)
	}
	available, err := gameWorld.GetAvailableQuests("Alice")
	if err != nil {
		t.Fatalf("GetAvailableQuests() returned an error: %v", err)
	}
	availableQuest := false
	for _, quest := range available {
		if quest.ID == "quest.explore_arvendale" {
			availableQuest = true
		}
	}
	if !availableQuest {
		t.Fatalf("available quests = %+v, want explore quest", available)
	}
	if err := gameWorld.AcceptQuest("Alice", "quest.explore_arvendale"); err != nil {
		t.Fatalf("AcceptQuest() returned an error: %v", err)
	}
	summary, err := gameWorld.GetPlayerQuestSummary("Alice", "quest.explore_arvendale")
	if err != nil {
		t.Fatalf("GetPlayerQuestSummary() returned an error: %v", err)
	}
	if summary.Status != world.QuestStatusActive || summary.Progress != "0/1" {
		t.Fatalf("active quest summary = %+v, want active at 0/1", summary)
	}
	quests, err := gameWorld.GetPlayerQuests("Alice")
	if err != nil {
		t.Fatalf("GetPlayerQuests() returned an error: %v", err)
	}
	if len(quests) != 1 || quests[0].ID != "quest.explore_arvendale" {
		t.Fatalf("player quests = %+v", quests)
	}

	if _, err := gameWorld.MovePlayer("Alice", "north"); err != nil {
		t.Fatalf("MovePlayer(town gate) returned an error: %v", err)
	}
	if err := gameWorld.TakeItem("Alice", "item.explorer_token"); err != nil {
		t.Fatalf("TakeItem(explorer token) returned an error: %v", err)
	}
	summary, err = gameWorld.GetPlayerQuestSummary("Alice", "quest.explore_arvendale")
	if err != nil {
		t.Fatalf("GetPlayerQuestSummary() after collecting objective returned an error: %v", err)
	}
	if summary.Progress != "1/1" {
		t.Fatalf("quest progress after collecting objective = %q, want 1/1", summary.Progress)
	}
	if _, err := gameWorld.MovePlayer("Alice", "south"); err != nil {
		t.Fatalf("MovePlayer(market street) returned an error: %v", err)
	}
	if err := gameWorld.CompleteQuest("Alice", "quest.explore_arvendale"); err != nil {
		t.Fatalf("CompleteQuest() returned an error: %v", err)
	}
	progress, err := gameWorld.GetPlayerQuest("Alice", "quest.explore_arvendale")
	if err != nil || progress.Status != world.QuestStatusCompleted {
		t.Fatalf("GetPlayerQuest() = (%+v, %v), want completed", progress, err)
	}
	quests, err = gameWorld.GetPlayerQuests("Alice")
	if err != nil || len(quests) != 1 || quests[0].Status != world.QuestStatusCompleted {
		t.Fatalf("GetPlayerQuests() after completion = (%+v, %v)", quests, err)
	}
	inventory, err := gameWorld.GetInventoryWithQuantities("Alice")
	if err != nil {
		t.Fatalf("GetInventoryWithQuantities() after completion returned an error: %v", err)
	}
	rewards := make(map[string]int)
	for _, item := range inventory {
		rewards[item.ID] = item.Quantity
	}
	if rewards["item.leather_boots"] != 1 || rewards["item.explorer_token"] != 0 {
		t.Fatalf("inventory after quest completion = %+v", inventory)
	}
}

func TestWorldQuestGiverAdvancesQuestChainAndRediscoversRepeatableQuest(t *testing.T) {
	gameWorld := loadTestWorld(t)
	if err := gameWorld.AddPlayer("Alice", "apothecary"); err != nil {
		t.Fatalf("AddPlayer(Alice) returned an error: %v", err)
	}

	conversation, err := gameWorld.TalkToNPC("Alice", "herbalist")
	if err != nil {
		t.Fatalf("TalkToNPC(herbalist) returned an error: %v", err)
	}
	if conversation.QuestStatus != world.QuestStatusAvailable ||
		conversation.Quest == nil || conversation.Quest.ID != "quest.gather_herbs" {
		t.Fatalf("initial herbalist conversation = %+v, want first quest available", conversation)
	}
	if err := gameWorld.AcceptQuest("Alice", "quest.gather_herbs"); err != nil {
		t.Fatalf("AcceptQuest(gather herbs) returned an error: %v", err)
	}
	conversation, err = gameWorld.TalkToNPC("Alice", "herbalist")
	if err != nil {
		t.Fatalf("TalkToNPC(active herb quest) returned an error: %v", err)
	}
	if conversation.QuestStatus != world.QuestStatusActive {
		t.Fatalf("active herb quest conversation status = %q", conversation.QuestStatus)
	}

	for _, direction := range []string{"west", "north", "north"} {
		if _, err := gameWorld.MovePlayer("Alice", direction); err != nil {
			t.Fatalf("MovePlayer(%s) toward forest returned an error: %v", direction, err)
		}
	}
	if err := gameWorld.TakeItem("Alice", "item.herbs"); err != nil {
		t.Fatalf("TakeItem(first herbs) returned an error: %v", err)
	}
	for _, direction := range []string{"south", "south", "east"} {
		if _, err := gameWorld.MovePlayer("Alice", direction); err != nil {
			t.Fatalf("MovePlayer(%s) back to apothecary returned an error: %v", direction, err)
		}
	}
	conversation, err = gameWorld.TalkToNPC("Alice", "herbalist")
	if err != nil {
		t.Fatalf("TalkToNPC(completed first quest) returned an error: %v", err)
	}
	if conversation.QuestStatus != world.QuestStatusAvailable ||
		conversation.Quest == nil || conversation.Quest.ID != "quest.repeatable_herbs" {
		t.Fatalf("quest-chain conversation = %+v, want repeatable quest available", conversation)
	}
	if err := gameWorld.AcceptQuest("Alice", "quest.repeatable_herbs"); err != nil {
		t.Fatalf("AcceptQuest(repeatable herbs) returned an error: %v", err)
	}
	conversation, err = gameWorld.TalkToNPC("Alice", "herbalist")
	if err != nil {
		t.Fatalf("TalkToNPC(active repeatable quest) returned an error: %v", err)
	}
	if conversation.QuestStatus != world.QuestStatusActive {
		t.Fatalf("active repeatable quest conversation status = %q", conversation.QuestStatus)
	}

	for _, direction := range []string{"west", "north", "north"} {
		if _, err := gameWorld.MovePlayer("Alice", direction); err != nil {
			t.Fatalf("MovePlayer(%s) toward forest returned an error: %v", direction, err)
		}
	}
	if err := gameWorld.TakeItem("Alice", "item.herbs"); err != nil {
		t.Fatalf("TakeItem(second herbs) returned an error: %v", err)
	}
	for _, direction := range []string{"south", "south", "east"} {
		if _, err := gameWorld.MovePlayer("Alice", direction); err != nil {
			t.Fatalf("MovePlayer(%s) back to apothecary returned an error: %v", direction, err)
		}
	}
	conversation, err = gameWorld.TalkToNPC("Alice", "herbalist")
	if err != nil {
		t.Fatalf("TalkToNPC(completed repeatable quest) returned an error: %v", err)
	}
	if conversation.QuestStatus != world.QuestStatusAvailable ||
		conversation.Quest == nil || conversation.Quest.ID != "quest.repeatable_herbs" {
		t.Fatalf("repeatable quest conversation = %+v, want quest rediscovered", conversation)
	}
}

func TestWorldCombatEffectsAndDefendAction(t *testing.T) {
	gameWorld := loadTestWorld(t)
	if err := gameWorld.AddPlayer("Alice", "docks"); err != nil {
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
	if combat.Participants[combat.TurnIndex].ID != "Alice" {
		t.Fatalf("initial combat turn = %q, want Alice", combat.Participants[combat.TurnIndex].ID)
	}

	effect := world.CombatEffect{
		ID:             "test-defense",
		Name:           "Test Defense",
		TargetID:       "Alice",
		Stat:           world.CombatStatDefense,
		Modifier:       3,
		Duration:       2,
		RemainingTurns: 2,
	}
	combat, err = gameWorld.ApplyCombatEffect(combat.ID, "Alice", effect)
	if err != nil {
		t.Fatalf("ApplyCombatEffect() returned an error: %v", err)
	}
	if len(combat.Participants[0].Effects) != 1 {
		t.Fatalf("effects after applying = %+v, want one effect", combat.Participants[0].Effects)
	}
	combat, err = gameWorld.RemoveCombatEffect(combat.ID, "Alice", effect.ID)
	if err != nil {
		t.Fatalf("RemoveCombatEffect() returned an error: %v", err)
	}
	if len(combat.Participants[0].Effects) != 0 {
		t.Fatalf("effects after removal = %+v, want none", combat.Participants[0].Effects)
	}

	combat, turnEvent, err := gameWorld.DefendCombatParticipant(combat.ID, "Alice")
	if err != nil {
		t.Fatalf("DefendCombatParticipant() returned an error: %v", err)
	}
	if len(combat.Participants[0].Effects) != 1 ||
		combat.Participants[0].Effects[0].ModifierPercent != world.DefaultDefenseBonusPercent {
		t.Fatalf("effects after defending = %+v, want temporary defense bonus", combat.Participants[0].Effects)
	}
	if turnEvent == nil || turnEvent.IsPlayer || turnEvent.ParticipantID != "dock_warden" {
		t.Fatalf("turn event after defend = %+v, want Dock Warden", turnEvent)
	}
}

func loadTestWorld(t *testing.T) *world.World {
	t.Helper()

	gameWorld, err := world.Load("../../../data/world.json")
	if err != nil {
		t.Fatalf("world.Load() returned an error: %v", err)
	}
	return gameWorld
}
