package internal_test

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"the-answer-protocol/internal/world"
)

func TestQuestDialogueMatchesQuestChainAndWorldJSON(t *testing.T) {
	raw, err := os.ReadFile("../../../data/world.json")
	if err != nil {
		t.Fatalf("read world.json: %v", err)
	}

	var definition struct {
		NPCs map[string]struct {
			QuestDialogues map[string]world.NPCDialogue `json:"quest_dialogues"`
		} `json:"npcs"`
	}
	if err := json.Unmarshal(raw, &definition); err != nil {
		t.Fatalf("decode world.json: %v", err)
	}

	gameWorld, err := world.Load("../../../data/world.json")
	if err != nil {
		t.Fatalf("world.Load(): %v", err)
	}
	if _, err := gameWorld.AddPlayer("Alice", "loc.apothecary_shop"); err != nil {
		t.Fatalf("AddPlayer(): %v", err)
	}

	assertDialogue := func(step, npcID, questID string, status world.QuestStatus) {
		t.Helper()
		conversation, err := gameWorld.TalkToNPC("Alice", npcID)
		if err != nil {
			t.Fatalf("%s: TalkToNPC(): %v", step, err)
		}
		if conversation.QuestStatus != status {
			t.Fatalf("%s: quest status = %q, want %q", step, conversation.QuestStatus, status)
		}
		if conversation.Quest == nil || conversation.Quest.ID != questID {
			t.Fatalf("%s: quest = %+v, want %q", step, conversation.Quest, questID)
		}

		dialogues, exists := definition.NPCs[npcID]
		if !exists {
			t.Fatalf("%s: NPC %q missing from world.json", step, npcID)
		}
		questDialogue, exists := dialogues.QuestDialogues[questID]
		if !exists {
			t.Fatalf("%s: dialogue for quest %q missing from world.json", step, questID)
		}

		var want []string
		switch status {
		case world.QuestStatusAvailable:
			want = questDialogue.Available
		case world.QuestStatusActive:
			want = questDialogue.Active
		case world.QuestStatusCompleted:
			want = questDialogue.Completed
		case world.QuestStatusLocked:
			want = questDialogue.Locked
		}
		if len(want) == 0 {
			want = questDialogue.Default
		}
		if !reflect.DeepEqual(conversation.Dialogue, want) {
			t.Fatalf("%s: dialogue = %v, want world.json %q dialogue %v", step, conversation.Dialogue, status, want)
		}
	}

	assertDialogue("initial", "npc.apprentice_lina", "quest.remedies_of_the_whispering_woods", world.QuestStatusAvailable)
	if err := gameWorld.AcceptQuest("Alice", "quest.remedies_of_the_whispering_woods"); err != nil {
		t.Fatalf("AcceptQuest(gather herbs): %v", err)
	}
	assertDialogue("accepted", "npc.apprentice_lina", "quest.remedies_of_the_whispering_woods", world.QuestStatusActive)

	for _, direction := range []string{"west", "north", "north"} {
		if _, err := gameWorld.MovePlayer("Alice", direction); err != nil {
			t.Fatalf("MovePlayer(%s): %v", direction, err)
		}
	}
	if err := gameWorld.TakeItem("Alice", "item.healing_herbs"); err != nil {
		t.Fatalf("TakeItem(herbs): %v", err)
	}
	for _, direction := range []string{"south", "south", "east"} {
		if _, err := gameWorld.MovePlayer("Alice", direction); err != nil {
			t.Fatalf("MovePlayer(%s): %v", direction, err)
		}
	}

	assertDialogue("first quest validated", "npc.apprentice_lina", "quest.remedies_of_the_whispering_woods", world.QuestStatusCompleted)
	assertDialogue("next quest offered", "npc.apprentice_lina", "quest.alchemists_herb_supply", world.QuestStatusAvailable)
}
