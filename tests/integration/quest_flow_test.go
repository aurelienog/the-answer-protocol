package internal_test

import (
	"encoding/json"
	"strings"
	"testing"

	"the-answer-protocol/internal/world"
)

func TestHandleConnectionQuestAwardsRewardAndConsumesObjective(t *testing.T) {
	alice := newConnectionTestEnv(t)
	connectTestPlayer(t, alice, "Alice")

	alice.sendRequest(t, "MOVE north")
	if got := alice.readResponse(t); got != "OK room=loc.market_street" {
		t.Fatalf("move to Quest Advisor response = %q", got)
	}

	alice.sendRequest(t, "QUEST Quest Advisor")
	questResponse := alice.readResponse(t)
	if !strings.HasPrefix(questResponse, "OK ") {
		t.Fatalf("quest acceptance response = %q, want OK JSON response", questResponse)
	}

	progress, err := alice.gameWorld.GetPlayerQuest("Alice", "quest.scout_the_realm_of_arvendale")
	if err != nil {
		t.Fatalf("GetPlayerQuest() returned an error: %v", err)
	}
	if progress.Status != world.QuestStatusActive {
		t.Fatalf("quest status after acceptance = %q, want %q", progress.Status, world.QuestStatusActive)
	}

	alice.sendRequest(t, "MOVE north")
	if got := alice.readResponse(t); got != "OK room=loc.north_town_gate" {
		t.Fatalf("move to town gate response = %q", got)
	}
	alice.sendRequest(t, "TAKE Northern Gate Seal")
	if got := alice.readResponse(t); got != "OK taken=item.northern_gate_seal" {
		t.Fatalf("take objective response = %q", got)
	}

	alice.sendRequest(t, "MOVE south")
	if got := alice.readResponse(t); got != "OK room=loc.market_street" {
		t.Fatalf("return to Quest Advisor response = %q", got)
	}
	alice.sendRequest(t, "TALK Quest Advisor")
	conversationResponse := alice.readResponse(t)
	if conversationResponse == "EVT QUEST COMPLETED quest=quest.scout_the_realm_of_arvendale" {
		conversationResponse = alice.readResponse(t)
	}
	if !strings.HasPrefix(conversationResponse, "OK ") {
		t.Fatalf("quest completion conversation = %q, want OK JSON response", conversationResponse)
	}

	var conversation struct {
		QuestStatus world.QuestStatus   `json:"quest_status"`
		Quest       *world.QuestSummary `json:"quest,omitempty"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(conversationResponse, "OK ")), &conversation); err != nil {
		t.Fatalf("decode quest completion conversation: %v; response: %q", err, conversationResponse)
	}
	if conversation.QuestStatus != world.QuestStatusCompleted {
		t.Fatalf("conversation quest status = %q, want %q", conversation.QuestStatus, world.QuestStatusCompleted)
	}

	progress, err = alice.gameWorld.GetPlayerQuest("Alice", "quest.scout_the_realm_of_arvendale")
	if err != nil {
		t.Fatalf("GetPlayerQuest() after completion returned an error: %v", err)
	}
	if progress.Status != world.QuestStatusCompleted {
		t.Fatalf("final quest status = %q, want %q", progress.Status, world.QuestStatusCompleted)
	}

	inventory, err := alice.gameWorld.GetInventory("Alice")
	if err != nil {
		t.Fatalf("GetInventory() returned an error: %v", err)
	}
	itemCounts := make(map[string]int, len(inventory))
	for _, item := range inventory {
		itemCounts[item.ID]++
	}
	if got := itemCounts["item.pathfinder_boots"]; got != 1 {
		t.Errorf("Pathfinder Boots quantity = %d, want 1", got)
	}
	if got := itemCounts["item.northern_gate_seal"]; got != 0 {
		t.Errorf("Northern Gate Seal quantity = %d after completion, want 0", got)
	}
}

func TestHandleConnectionAdvancesQuestChainAfterObjectiveDelivery(t *testing.T) {
	alice := newConnectionTestEnv(t)
	connectTestPlayer(t, alice, "Alice")

	alice.sendRequest(t, "QUEST Captain Eliza")
	if got := alice.readResponse(t); !strings.HasPrefix(got, "OK ") {
		t.Fatalf("quest acceptance response = %q, want OK JSON response", got)
	}

	alice.sendRequest(t, "TALK Captain Eliza")
	conversation := readQuestConversation(t, alice)
	if conversation.QuestStatus != world.QuestStatusActive ||
		conversation.Quest == nil ||
		conversation.Quest.Progress != "0/1" {
		t.Fatalf("quest before collecting objective = %+v, want active at 0/1", conversation)
	}
	progress, err := alice.gameWorld.GetPlayerQuest("Alice", "quest.expose_the_harbor_smugglers")
	if err != nil {
		t.Fatalf("GetPlayerQuest() before objective delivery returned an error: %v", err)
	}
	if progress.Status != world.QuestStatusActive {
		t.Fatalf("quest status before objective delivery = %q, want %q", progress.Status, world.QuestStatusActive)
	}
	inventory, err := alice.gameWorld.GetInventory("Alice")
	if err != nil {
		t.Fatalf("GetInventory() before objective delivery returned an error: %v", err)
	}
	for _, item := range inventory {
		if item.ID == "item.royal_guard_breastplate" {
			t.Fatal("quest reward was granted before objective delivery")
		}
	}

	alice.sendRequest(t, "MOVE south")
	if got := alice.readResponse(t); got != "OK room=loc.the_great_pier" {
		t.Fatalf("move to loc.the_great_pier response = %q", got)
	}
	alice.sendRequest(t, "MOVE east")
	if got := alice.readResponse(t); got != "OK room=loc.abandoned_warehouse" {
		t.Fatalf("move to loc.abandoned_warehouse response = %q", got)
	}
	alice.sendRequest(t, "TAKE Incriminating Letter")
	if got := alice.readResponse(t); got != "OK taken=item.incriminating_letter" {
		t.Fatalf("take quest objective response = %q", got)
	}

	alice.sendRequest(t, "MOVE west")
	if got := alice.readResponse(t); got != "OK room=loc.the_great_pier" {
		t.Fatalf("return to loc.the_great_pier response = %q", got)
	}
	alice.sendRequest(t, "MOVE north")
	if got := alice.readResponse(t); got != "OK room=loc.central_harbor_square" {
		t.Fatalf("return to Captain Eliza response = %q", got)
	}

	alice.sendRequest(t, "TALK Captain Eliza")
	conversation = readQuestConversation(t, alice)
	if conversation.QuestStatus != world.QuestStatusCompleted ||
		conversation.Quest == nil ||
		conversation.Quest.ID != "quest.expose_the_harbor_smugglers" {
		t.Fatalf("quest immediately after objective delivery = %+v, want completed quest dialogue", conversation)
	}

	alice.sendRequest(t, "TALK Captain Eliza")
	conversation = readQuestConversation(t, alice)
	if conversation.QuestStatus != world.QuestStatusAvailable ||
		conversation.Quest == nil ||
		conversation.Quest.ID != "quest.the_vanishing_fleet" {
		t.Fatalf("quest after objective delivery = %+v, want next quest available", conversation)
	}

	progress, err = alice.gameWorld.GetPlayerQuest("Alice", "quest.expose_the_harbor_smugglers")
	if err != nil {
		t.Fatalf("GetPlayerQuest() returned an error: %v", err)
	}
	if progress.Status != world.QuestStatusCompleted {
		t.Fatalf("first quest status = %q, want %q", progress.Status, world.QuestStatusCompleted)
	}

	inventory, err = alice.gameWorld.GetInventory("Alice")
	if err != nil {
		t.Fatalf("GetInventory() returned an error: %v", err)
	}
	itemCounts := make(map[string]int, len(inventory))
	for _, item := range inventory {
		itemCounts[item.ID]++
	}
	if itemCounts["item.incriminating_letter"] != 0 {
		t.Errorf("Incriminating Letter quantity after delivery = %d, want 0", itemCounts["item.incriminating_letter"])
	}
	if itemCounts["item.royal_guard_breastplate"] != 1 {
		t.Errorf("Royal Guard Breastplate quantity = %d, want 1", itemCounts["item.royal_guard_breastplate"])
	}

	alice.sendRequest(t, "TALK Captain Eliza")
	conversation = readQuestConversation(t, alice)
	if conversation.QuestStatus != world.QuestStatusAvailable ||
		conversation.Quest == nil ||
		conversation.Quest.ID != "quest.the_vanishing_fleet" {
		t.Fatalf("quest after repeated talk = %+v, want next quest still available", conversation)
	}
	inventory, err = alice.gameWorld.GetInventory("Alice")
	if err != nil {
		t.Fatalf("GetInventory() after repeated talk returned an error: %v", err)
	}
	rewardCount := 0
	for _, item := range inventory {
		if item.ID == "item.royal_guard_breastplate" {
			rewardCount++
		}
	}
	if rewardCount != 1 {
		t.Fatalf("Royal Guard Breastplate quantity after repeated talk = %d, want 1", rewardCount)
	}

	alice.sendRequest(t, "QUEST Captain Eliza")
	if got := alice.readResponse(t); !strings.HasPrefix(got, "OK ") {
		t.Fatalf("next quest acceptance response = %q, want OK JSON response", got)
	}
	progress, err = alice.gameWorld.GetPlayerQuest("Alice", "quest.the_vanishing_fleet")
	if err != nil {
		t.Fatalf("GetPlayerQuest(next quest) returned an error: %v", err)
	}
	if progress.Status != world.QuestStatusActive {
		t.Errorf("next quest status = %q, want %q", progress.Status, world.QuestStatusActive)
	}
}

func readQuestConversation(t *testing.T, env *connectionTestEnv) struct {
	QuestStatus world.QuestStatus   `json:"quest_status"`
	Quest       *world.QuestSummary `json:"quest,omitempty"`
} {
	t.Helper()

	response := env.readResponse(t)
	for strings.HasPrefix(response, "EVT QUEST COMPLETED ") {
		response = env.readResponse(t)
	}
	if !strings.HasPrefix(response, "OK ") {
		t.Fatalf("quest conversation = %q, want OK JSON response", response)
	}

	var conversation struct {
		QuestStatus world.QuestStatus   `json:"quest_status"`
		Quest       *world.QuestSummary `json:"quest,omitempty"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(response, "OK ")), &conversation); err != nil {
		t.Fatalf("decode quest conversation: %v; response: %q", err, response)
	}
	return conversation
}
