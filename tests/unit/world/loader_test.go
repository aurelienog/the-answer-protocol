package internal_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"the-answer-protocol/internal/world"
)

func TestLoadWorld(t *testing.T) {
	gameWorld, err := world.Load("../../../data/world.json")
	if err != nil {
		t.Fatalf("world.Load() returned an error: %v", err)
	}

	if gameWorld == nil {
		t.Fatal("world.Load() returned a nil world without an error")
	}

	if got := gameWorld.RoomCount(); got != 9 {
		t.Fatalf("expected 9 rooms, got %d", got)
	}

	room, err := gameWorld.GetRoom("harbor_square")
	if err != nil {
		t.Fatalf("expected harbor_square to exist: %v", err)
	}

	if room.ID != "harbor_square" {
		t.Errorf("expected room ID %q, got %q", "harbor_square", room.ID)
	}

	if room.Name != "Central Harbor Square" {
		t.Errorf("unexpected room name: %q", room.Name)
	}

	if got := room.Exits["north"]; got != "market_street" {
		t.Errorf("expected north exit to market_street, got %q", got)
	}

	items, err := gameWorld.GetRoomItems("harbor_square")
	if err != nil {
		t.Fatalf("expected to get items in harbor_square: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item in harbor_square, got %d", len(items))
	}

	item := items[0]
	if item.ID != "item.rope" {
		t.Errorf("expected item ID %q, got %q", "item.rope", item.ID)
	}

	if item.Name != "Hempen Rope" {
		t.Errorf("expected item name %q, got %q", "Hempen Rope", item.Name)
	}

	if item.Description == "" {
		t.Error("expected item description to be non-empty")
	}

	if item.Type != world.ItemTypeCommon {
		t.Errorf(
			"expected item type %q, got %q",
			world.ItemTypeCommon,
			item.Type,
		)
	}
}

func TestLoadWorldPopulatesCurrentRooms(t *testing.T) {
	gameWorld, err := world.Load("../../../data/world.json")
	if err != nil {
		t.Fatalf("world.Load() returned an error: %v", err)
	}

	tests := []struct {
		roomID string
		name   string
		items  map[string]int
		npcIDs []string
	}{
		{
			roomID: "harbor_square",
			name:   "Central Harbor Square",
			items:  map[string]int{"item.rope": 1},
			npcIDs: []string{"captain_eliza"},
		},
		{
			roomID: "market_street",
			name:   "Market Street",
			items:  map[string]int{"item.pouch": 2},
			npcIDs: []string{"town_guide", "quest_advisor"},
		},
		{
			roomID: "blacksmith_forge",
			name:   "Royal Blacksmith Forge",
			items: map[string]int{
				"item.iron_sword": 1,
				"item.oak_shield": 1,
			},
			npcIDs: []string{"blacksmith"},
		},
		{
			roomID: "apothecary",
			name:   "Apothecary Shop",
			items:  map[string]int{"item.healing_potion": 3},
			npcIDs: []string{"herbalist"},
		},
		{
			roomID: "tavern",
			name:   "The Salty Siren Tavern",
			items: map[string]int{
				"item.bread": 3,
				"item.wine":  2,
			},
			npcIDs: []string{"tavern_keeper"},
		},
		{
			roomID: "docks",
			name:   "The Great Pier",
			items: map[string]int{
				"item.lantern":              2,
				"item.missing_ships_report": 1,
			},
			npcIDs: []string{"dock_warden"},
		},
		{
			roomID: "warehouse",
			name:   "Abandoned Warehouse",
			items:  map[string]int{"item.compromise_letter": 1},
			npcIDs: []string{"smuggler"},
		},
		{
			roomID: "town_gate",
			name:   "North Town Gate",
			items:  map[string]int{"item.explorer_token": 2},
			npcIDs: []string{"gate_guard"},
		},
		{
			roomID: "forest_edge",
			name:   "Whispering Forest Edge",
			items:  map[string]int{"item.herbs": 2},
			npcIDs: []string{"dire_wolf"},
		},
	}

	totalItemInstances := 0
	for _, tt := range tests {
		t.Run(tt.roomID, func(t *testing.T) {
			room, err := gameWorld.GetRoom(tt.roomID)
			if err != nil {
				t.Fatalf("GetRoom(%q) returned an error: %v", tt.roomID, err)
			}

			if room.Name != tt.name {
				t.Errorf("expected room name %q, got %q", tt.name, room.Name)
			}

			if !reflect.DeepEqual(room.Items, tt.items) {
				t.Errorf("expected room items %#v, got %#v", tt.items, room.Items)
			}

			roomItems, err := gameWorld.GetRoomItems(tt.roomID)
			if err != nil {
				t.Fatalf("GetRoomItems(%q) returned an error: %v", tt.roomID, err)
			}

			itemCounts := make(map[string]int, len(roomItems))
			for _, item := range roomItems {
				itemCounts[item.ID]++
			}

			if !reflect.DeepEqual(itemCounts, tt.items) {
				t.Errorf("expected item instances %#v, got %#v", tt.items, itemCounts)
			}

			npcs, err := gameWorld.GetNPCsInRoom(tt.roomID)
			if err != nil {
				t.Fatalf("GetNPCsInRoom(%q) returned an error: %v", tt.roomID, err)
			}

			gotNPCIDs := make([]string, len(npcs))
			for i, npc := range npcs {
				gotNPCIDs[i] = npc.ID
			}

			if !reflect.DeepEqual(gotNPCIDs, tt.npcIDs) {
				t.Errorf("expected NPCs %#v, got %#v", tt.npcIDs, gotNPCIDs)
			}
		})

		for _, quantity := range tt.items {
			totalItemInstances += quantity
		}
	}

	if totalItemInstances != 21 {
		t.Errorf("expected 21 item instances across rooms, got %d", totalItemInstances)
	}
}

func TestLoadWorldRejectsInvalidDefinitions(t *testing.T) {
	tests := []struct {
		name            string
		definition      string
		expectedInError string
	}{
		{
			name:            "empty world",
			definition:      `{"rooms":{}}`,
			expectedInError: "world has no rooms",
		},
		{
			name: "invalid room exit",
			definition: `{"rooms":{"room":{"id":"room","name":"Room",` +
				`"exits":{"north":"missing"}}}}`,
			expectedInError: `exit "north" to nonexistent room "missing"`,
		},
		{
			name: "room references unknown item",
			definition: `{"rooms":{"room":{"id":"room","name":"Room",` +
				`"items":{"item.missing":1}}}}`,
			expectedInError: `references nonexistent item "item.missing"`,
		},
		{
			name: "room item quantity must be positive",
			definition: `{"rooms":{"room":{"id":"room","name":"Room",` +
				`"items":{"item.coin":0}}},"items":{"item.coin":` +
				`{"id":"item.coin","name":"Coin","type":"common"}}}`,
			expectedInError: `invalid quantity 0 for item "item.coin"`,
		},
		{
			name: "invalid item type",
			definition: `{"rooms":{"room":{"id":"room","name":"Room"}},` +
				`"items":{"item.coin":{"id":"item.coin","name":"Coin",` +
				`"type":"unknown"}}}`,
			expectedInError: `item "item.coin" has invalid type "unknown"`,
		},
		{
			name: "invalid NPC stats",
			definition: `{"rooms":{"room":{"id":"room","name":"Room"}},` +
				`"npcs":{"npc":{"id":"npc","name":"NPC",` +
				`"combat_stats":{"hp":1,"max_hp":0}}}}`,
			expectedInError: `NPC "npc" has invalid max HP 0`,
		},
		{
			name: "quest references unknown NPC",
			definition: `{"rooms":{"room":{"id":"room","name":"Room"}},` +
				`"quests":{"quest.test":{"id":"quest.test","name":"Test",` +
				`"giver_npc_id":"missing"}}}`,
			expectedInError: `references nonexistent giver NPC "missing"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeWorldDefinition(t, tt.definition)

			gameWorld, err := world.Load(path)
			if err == nil {
				t.Fatal("world.Load() succeeded for an invalid definition")
			}

			if gameWorld != nil {
				t.Error("world.Load() returned a world together with an error")
			}

			if !strings.Contains(err.Error(), tt.expectedInError) {
				t.Errorf(
					"expected error to contain %q, got %q",
					tt.expectedInError,
					err,
				)
			}
		})
	}
}

func TestLoadWorldReportsFileAndJSONErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := world.Load(filepath.Join(t.TempDir(), "missing.json"))
		if err == nil {
			t.Fatal("world.Load() succeeded for a missing file")
		}

		if !strings.Contains(err.Error(), "read world file") {
			t.Errorf("expected read error, got %q", err)
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		path := writeWorldDefinition(t, `{"rooms":`)
		_, err := world.Load(path)
		if err == nil {
			t.Fatal("world.Load() succeeded for malformed JSON")
		}

		if !strings.Contains(err.Error(), "decode world file") {
			t.Errorf("expected decode error, got %q", err)
		}
	})
}

func TestLoadWorldReturnsErrorsForUnknownRoom(t *testing.T) {
	gameWorld, err := world.Load("../../../data/world.json")
	if err != nil {
		t.Fatalf("world.Load() returned an error: %v", err)
	}

	if _, err := gameWorld.GetRoom("missing"); err == nil {
		t.Error("GetRoom() succeeded for an unknown room")
	}

	if _, err := gameWorld.GetRoomItems("missing"); err == nil {
		t.Error("GetRoomItems() succeeded for an unknown room")
	}

	if _, err := gameWorld.GetNPCsInRoom("missing"); err == nil {
		t.Error("GetNPCsInRoom() succeeded for an unknown room")
	}
}

func writeWorldDefinition(t *testing.T, definition string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "world.json")
	if err := os.WriteFile(path, []byte(definition), 0o600); err != nil {
		t.Fatalf("failed to write world definition: %v", err)
	}

	return path
}
