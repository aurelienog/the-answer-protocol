package internal_test

import (
	"encoding/json"
	"strings"
	"testing"

	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

func TestInventoryDropAndStatusCommands(t *testing.T) {
	gameWorld := loadCommandTestWorld(t)
	if err := gameWorld.AddPlayer("Alice", "harbor_square"); err != nil {
		t.Fatalf("AddPlayer(Alice) returned an error: %v", err)
	}
	ctx := &commands.Context{PlayerName: "Alice", World: gameWorld}

	inventory, err := commands.NewInventoryCommand().Execute(ctx)
	if err != nil || inventory.Kind != protocol.KindOK || inventory.Payload != "[]" {
		t.Fatalf("INVENTORY response = (%+v, %v), want empty inventory", inventory, err)
	}
	status, err := commands.NewStatusCommand().Execute(ctx)
	if err != nil || status.Kind != protocol.KindOK {
		t.Fatalf("STATUS response = (%+v, %v), want OK", status, err)
	}
	var statusPayload struct {
		HP     int    `json:"hp"`
		MaxHP  int    `json:"max_hp"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(status.Payload), &statusPayload); err != nil {
		t.Fatalf("decode STATUS response: %v; payload: %q", err, status.Payload)
	}
	if statusPayload.HP != statusPayload.MaxHP || statusPayload.Status != "healthy" {
		t.Fatalf("STATUS payload = %+v, want full healthy HP", statusPayload)
	}

	if err := gameWorld.TakeItem("Alice", "item.rope"); err != nil {
		t.Fatalf("TakeItem(rope) returned an error: %v", err)
	}
	ctx.RawText = "Hempen Rope"
	drop, err := commands.NewDropCommand().Execute(ctx)
	if err != nil || drop.Kind != protocol.KindOK || drop.Payload != "dropped=item.rope" {
		t.Fatalf("DROP response = (%+v, %v)", drop, err)
	}

	ctx.RawText = "Purse of Gold Coins"
	drop, err = commands.NewDropCommand().Execute(ctx)
	if err != nil || drop.Kind != protocol.KindERR ||
		drop.Code != protocol.CodeItemNotInInventory ||
		drop.Name != "ITEM_NOT_IN_INVENTORY" {
		t.Fatalf("DROP missing item response = (%+v, %v)", drop, err)
	}
}

func TestTalkQuestsAndCompleteCommands(t *testing.T) {
	gameWorld := loadCommandTestWorld(t)
	if err := gameWorld.AddPlayer("Alice", "market_street"); err != nil {
		t.Fatalf("AddPlayer(Alice) returned an error: %v", err)
	}
	ctx := &commands.Context{PlayerName: "Alice", World: gameWorld}

	ctx.RawArgs = []string{"Quest", "Advisor"}
	talk, err := commands.NewTalkCommand().Execute(ctx)
	if err != nil || talk.Kind != protocol.KindOK {
		t.Fatalf("TALK response = (%+v, %v), want OK", talk, err)
	}

	if err := gameWorld.AcceptQuest("Alice", "quest.explore_arvendale"); err != nil {
		t.Fatalf("AcceptQuest() returned an error: %v", err)
	}
	ctx.RawArgs = nil
	quests, err := commands.NewQuestsCommand().Execute(ctx)
	if err != nil || quests.Kind != protocol.KindOK ||
		!strings.Contains(quests.Payload, `"status":"active"`) {
		t.Fatalf("QUESTS response = (%+v, %v), want active quest JSON", quests, err)
	}

	ctx.RawArgs = []string{"quest.explore_arvendale"}
	complete, err := commands.NewCompleteCommand().Execute(ctx)
	if err != nil || complete.Kind != protocol.KindERR ||
		complete.Code != protocol.CodeCommandFailed ||
		complete.Name != "REQUIRED_ITEM_MISSING" {
		t.Fatalf("early COMPLETE response = (%+v, %v), want missing-item error", complete, err)
	}
	if progress, err := gameWorld.GetPlayerQuest("Alice", "quest.explore_arvendale"); err != nil ||
		progress.Status != world.QuestStatusActive {
		t.Fatalf("quest after failed COMPLETE = (%+v, %v), want active", progress, err)
	}

	ctx.RawArgs = []string{"unknown"}
	talk, err = commands.NewTalkCommand().Execute(ctx)
	if err != nil || talk.Kind != protocol.KindERR ||
		talk.Code != protocol.CodeNPCNotFound ||
		talk.Name != "NPC_NOT_FOUND" {
		t.Fatalf("TALK unknown NPC response = (%+v, %v)", talk, err)
	}
}

func TestAttackCommandMapsHostileAndTargetErrors(t *testing.T) {
	gameWorld := loadCommandTestWorld(t)
	if err := gameWorld.AddPlayer("Alice", "harbor_square"); err != nil {
		t.Fatalf("AddPlayer(Alice) returned an error: %v", err)
	}
	ctx := &commands.Context{PlayerName: "Alice", World: gameWorld}
	command := commands.NewAttackCommand()

	ctx.RawArgs = []string{"Captain", "Eliza"}
	response, err := command.Execute(ctx)
	if err != nil || response.Kind != protocol.KindERR ||
		response.Code != protocol.CodeNPCNotHostile ||
		response.Name != "NPC_NOT_HOSTILE" {
		t.Fatalf("ATTACK non-hostile NPC response = (%+v, %v)", response, err)
	}

	ctx.RawArgs = []string{"missing"}
	response, err = command.Execute(ctx)
	if err != nil || response.Kind != protocol.KindERR ||
		response.Code != protocol.CodeNPCNotFound ||
		response.Name != "NPC_NOT_FOUND" {
		t.Fatalf("ATTACK unknown NPC response = (%+v, %v)", response, err)
	}
}

func loadCommandTestWorld(t *testing.T) *world.World {
	t.Helper()

	gameWorld, err := world.Load("../../../data/world.json")
	if err != nil {
		t.Fatalf("world.Load() returned an error: %v", err)
	}
	return gameWorld
}
