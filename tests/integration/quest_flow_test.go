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
	if got := alice.readResponse(t); got != "OK room=market_street" {
		t.Fatalf("move to Quest Advisor response = %q", got)
	}

	alice.sendRequest(t, "QUEST Quest Advisor")
	questResponse := alice.readResponse(t)
	if !strings.HasPrefix(questResponse, "OK ") {
		t.Fatalf("quest acceptance response = %q, want OK JSON response", questResponse)
	}

	progress, err := alice.gameWorld.GetPlayerQuest("Alice", "quest.explore_arvendale")
	if err != nil {
		t.Fatalf("GetPlayerQuest() returned an error: %v", err)
	}
	if progress.Status != world.QuestStatusActive {
		t.Fatalf("quest status after acceptance = %q, want %q", progress.Status, world.QuestStatusActive)
	}

	alice.sendRequest(t, "MOVE north")
	if got := alice.readResponse(t); got != "OK room=town_gate" {
		t.Fatalf("move to town gate response = %q", got)
	}
	alice.sendRequest(t, "TAKE Northern Gate Seal")
	if got := alice.readResponse(t); got != "OK taken=item.explorer_token" {
		t.Fatalf("take objective response = %q", got)
	}

	alice.sendRequest(t, "MOVE south")
	if got := alice.readResponse(t); got != "OK room=market_street" {
		t.Fatalf("return to Quest Advisor response = %q", got)
	}
	alice.sendRequest(t, "TALK Quest Advisor")
	conversationResponse := alice.readResponse(t)
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

	progress, err = alice.gameWorld.GetPlayerQuest("Alice", "quest.explore_arvendale")
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
	if got := itemCounts["item.leather_boots"]; got != 1 {
		t.Errorf("Pathfinder Boots quantity = %d, want 1", got)
	}
	if got := itemCounts["item.explorer_token"]; got != 0 {
		t.Errorf("Northern Gate Seal quantity = %d after completion, want 0", got)
	}
}
