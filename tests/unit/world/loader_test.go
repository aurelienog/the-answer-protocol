package internal_test

import (
	"encoding/json"
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
			expectedInError: `exit "north" references unknown destination room "missing"`,
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
				`"combat_stats":{"hp":1,"max_hp":0,"attack":1,` +
				`"defense":0,"speed":1}}}}`,
			expectedInError: `NPC "npc" has invalid max HP 0`,
		},
		{
			name: "quest references unknown NPC",
			definition: `{"rooms":{"room":{"id":"room","name":"Room"}},` +
				`"quests":{"quest.test":{"id":"quest.test","name":"Test",` +
				`"giver_npc_id":"missing"}}}`,
			expectedInError: `references nonexistent giver NPC "missing"`,
		},
		{
			name: "quest is not in giver NPC's quest chain",
			definition: `{"rooms":{"harbor_square":{"id":"harbor_square",` +
				`"name":"Room","npcs":["npc"]}},"npcs":{"npc":{"id":"npc",` +
				`"name":"NPC","combat_stats":{"hp":1,"max_hp":1,"attack":1,` +
				`"defense":0,"speed":1},"quest_id":"quest.root"}},` +
				`"quests":{"quest.root":{"id":"quest.root","name":"Root",` +
				`"giver_npc_id":"npc"},"quest.other":{"id":"quest.other",` +
				`"name":"Other","giver_npc_id":"npc"}}}`,
			expectedInError: `does not expose quest "quest.other" in its quest chain`,
		},
		{
			name: "quest chain giver disagrees with existing NPC",
			definition: `{"rooms":{"harbor_square":{"id":"harbor_square",` +
				`"name":"Room","npcs":["npc.first","npc.second"]}},` +
				`"npcs":{"npc.first":{"id":"npc.first","name":"First",` +
				`"quest_id":"quest.first","combat_stats":{"hp":1,"max_hp":1,` +
				`"attack":1,"defense":0,"speed":1}},` +
				`"npc.second":{"id":"npc.second","name":"Second",` +
				`"combat_stats":{"hp":1,"max_hp":1,"attack":1,"defense":0,` +
				`"speed":1}}},"quests":{"quest.first":{"id":"quest.first",` +
				`"name":"First","giver_npc_id":"npc.first",` +
				`"next_quest_id":"quest.second"},"quest.second":{` +
				`"id":"quest.second","name":"Second",` +
				`"giver_npc_id":"npc.second"}}}`,
			expectedInError: `quest "quest.second" assigned to NPC "npc.second"`,
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

func TestLoadWorldRejectsMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name            string
		definition      string
		expectedInError string
	}{
		{
			name:            "missing rooms section",
			definition:      `{}`,
			expectedInError: "required field world.rooms is missing",
		},
		{
			name:            "missing room name",
			definition:      `{"rooms":{"harbor_square":{"id":"harbor_square"}}}`,
			expectedInError: `world.rooms["harbor_square"].name is missing or null`,
		},
		{
			name: "missing item type",
			definition: `{"rooms":{"harbor_square":{"id":"harbor_square",` +
				`"name":"Room"}},"items":{"item":{"id":"item","name":"Item"}}}`,
			expectedInError: `world.items["item"].type is missing or null`,
		},
		{
			name: "missing NPC combat statistic",
			definition: `{"rooms":{"harbor_square":{"id":"harbor_square",` +
				`"name":"Room"}},"npcs":{"npc":{"id":"npc","name":"NPC",` +
				`"combat_stats":{"hp":1,"max_hp":1,"attack":1,` +
				`"speed":1}}}}`,
			expectedInError: `world.npcs["npc"].combat_stats.defense is missing or null`,
		},
		{
			name: "missing quest giver",
			definition: `{"rooms":{"harbor_square":{"id":"harbor_square",` +
				`"name":"Room"}},"quests":{"quest":{"id":"quest",` +
				`"name":"Quest"}}}`,
			expectedInError: `world.quests["quest"].giver_npc_id is missing or null`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := world.Load(writeWorldDefinition(t, test.definition))
			if err == nil {
				t.Fatal("world.Load() succeeded with a missing required field")
			}
			if !strings.Contains(err.Error(), test.expectedInError) {
				t.Fatalf("error = %q, want it to contain %q", err, test.expectedInError)
			}
		})
	}
}

func TestLoadWorldRejectsAdditionalInvalidDefinitions(t *testing.T) {
	const validDefinition = `{
		"rooms":{"harbor_square":{"id":"harbor_square","name":"Room","npcs":["npc"],"exits":{},"items":{}}},
		"items":{"item":{"id":"item","name":"Item","type":"common"}},
		"npcs":{"npc":{"id":"npc","name":"NPC","quest_id":"quest","combat_stats":{"hp":1,"max_hp":1,"attack":1,"defense":0,"speed":1}}},
		"quests":{"quest":{"id":"quest","name":"Quest","giver_npc_id":"npc"}}
	}`
	tests := []struct {
		name            string
		path            []string
		value           interface{}
		expectedInError string
	}{
		{name: "room ID mismatch", path: []string{"rooms", "harbor_square", "id"}, value: "different", expectedInError: `does not match room ID`},
		{name: "empty room name", path: []string{"rooms", "harbor_square", "name"}, value: " ", expectedInError: `has an empty name`},
		{name: "invalid exit direction", path: []string{"rooms", "harbor_square", "exits"}, value: map[string]interface{}{"up": "harbor_square"}, expectedInError: `invalid exit direction "up"`},
		{name: "empty exit destination", path: []string{"rooms", "harbor_square", "exits"}, value: map[string]interface{}{"north": " "}, expectedInError: `empty destination for exit "north"`},
		{name: "unknown room NPC", path: []string{"rooms", "harbor_square", "npcs"}, value: []interface{}{"missing"}, expectedInError: `references unknown NPC "missing"`},
		{name: "duplicate room NPC", path: []string{"rooms", "harbor_square", "npcs"}, value: []interface{}{"npc", "npc"}, expectedInError: `contains duplicate NPC "npc"`},
		{name: "empty room NPC", path: []string{"rooms", "harbor_square", "npcs"}, value: []interface{}{""}, expectedInError: `contains an empty NPC ID`},
		{name: "unknown room item", path: []string{"rooms", "harbor_square", "items"}, value: map[string]interface{}{"missing": 1}, expectedInError: `references nonexistent item "missing"`},
		{name: "negative room item quantity", path: []string{"rooms", "harbor_square", "items"}, value: map[string]interface{}{"item": -1}, expectedInError: `invalid quantity -1 for item "item"`},
		{name: "multiple unique items in room", path: []string{"rooms", "harbor_square", "items"}, value: map[string]interface{}{"item": 2}, expectedInError: `contains multiple instances of unique item "item"`},
		{name: "unique item in multiple rooms", path: []string{"rooms"}, value: map[string]interface{}{"harbor_square": map[string]interface{}{"id": "harbor_square", "name": "Room", "npcs": []interface{}{"npc"}, "items": map[string]interface{}{"item": 1}}, "other": map[string]interface{}{"id": "other", "name": "Other", "items": map[string]interface{}{"item": 1}}}, expectedInError: `duplicate item instance ID`},
		{name: "item ID mismatch", path: []string{"items", "item", "id"}, value: "different", expectedInError: `does not match item ID`},
		{name: "empty item name", path: []string{"items", "item", "name"}, value: "", expectedInError: `has an empty name`},
		{name: "invalid item type", path: []string{"items", "item", "type"}, value: "invalid", expectedInError: `has invalid type "invalid"`},
		{name: "negative item combat attribute", path: []string{"items", "item", "attack"}, value: -1, expectedInError: `has negative combat attributes`},
		{name: "equipment without attributes", path: []string{"items", "item", "type"}, value: "equipment", expectedInError: `has no combat attributes`},
		{name: "empty NPC ID", path: []string{"npcs", "npc", "id"}, value: "", expectedInError: `NPC ID cannot be empty`},
		{name: "negative NPC HP", path: []string{"npcs", "npc", "combat_stats", "hp"}, value: -1, expectedInError: `has negative HP -1`},
		{name: "NPC HP exceeds maximum", path: []string{"npcs", "npc", "combat_stats", "hp"}, value: 2, expectedInError: `has HP 2 greater than max HP 1`},
		{name: "NPC attack is zero", path: []string{"npcs", "npc", "combat_stats", "attack"}, value: 0, expectedInError: `invalid attack 0`},
		{name: "negative NPC defense", path: []string{"npcs", "npc", "combat_stats", "defense"}, value: -1, expectedInError: `negative defense -1`},
		{name: "invalid NPC speed", path: []string{"npcs", "npc", "combat_stats", "speed"}, value: 0, expectedInError: `invalid speed 0`},
		{name: "empty NPC name", path: []string{"npcs", "npc", "name"}, value: " ", expectedInError: `has an empty name`},
		{name: "unknown NPC reward", path: []string{"npcs", "npc", "rewards"}, value: []interface{}{"missing"}, expectedInError: `references nonexistent reward item "missing"`},
		{name: "empty NPC reward", path: []string{"npcs", "npc", "rewards"}, value: []interface{}{""}, expectedInError: `has an empty reward item ID`},
		{name: "empty quest name", path: []string{"quests", "quest", "name"}, value: "", expectedInError: `has an empty name`},
		{name: "quest quantity without item", path: []string{"quests", "quest", "required_item_quantity"}, value: 1, expectedInError: `has required item quantity 1 but no required item ID`},
		{name: "quest item quantity is zero", path: []string{"quests", "quest", "required_item_id"}, value: "item", expectedInError: `invalid required item quantity 0`},
		{name: "unknown quest item", path: []string{"quests", "quest", "required_item_id"}, value: "missing", expectedInError: `references nonexistent required item "missing"`},
		{name: "empty quest reward", path: []string{"quests", "quest", "reward_item_ids"}, value: []interface{}{""}, expectedInError: `has an empty reward item ID`},
		{name: "unknown quest reward", path: []string{"quests", "quest", "reward_item_ids"}, value: []interface{}{"missing"}, expectedInError: `references nonexistent reward item "missing"`},
		{name: "empty quest giver", path: []string{"quests", "quest", "giver_npc_id"}, value: "", expectedInError: `has an empty giver NPC ID`},
		{name: "unknown next quest", path: []string{"quests", "quest", "next_quest_id"}, value: "missing", expectedInError: `references nonexistent next quest "missing"`},
		{name: "quest chain cycle", path: []string{"quests", "quest", "next_quest_id"}, value: "quest", expectedInError: `quest chain contains a cycle`},
		{name: "unknown quest dialogue", path: []string{"npcs", "npc", "quest_dialogues"}, value: map[string]interface{}{"missing": map[string]interface{}{}}, expectedInError: `references nonexistent quest "missing" in quest dialogues`},
		{name: "NPC references unknown quest", path: []string{"npcs", "npc", "quest_id"}, value: "missing", expectedInError: `references nonexistent quest "missing"`},
		{name: "quest giver disagrees with NPC chain", path: []string{"quests", "quest", "giver_npc_id"}, value: "other", expectedInError: `references nonexistent giver NPC "other"`},
		{name: "NPC has no room", path: []string{"rooms", "harbor_square", "npcs"}, value: []interface{}{}, expectedInError: `is not assigned to any room`},
		{name: "unknown exit destination", path: []string{"rooms", "harbor_square", "exits"}, value: map[string]interface{}{"north": "missing"}, expectedInError: `references unknown destination room "missing"`},
		{name: "room disconnected from initial room", path: []string{"rooms"}, value: map[string]interface{}{"harbor_square": map[string]interface{}{"id": "harbor_square", "name": "Room", "npcs": []interface{}{"npc"}}, "isolated": map[string]interface{}{"id": "isolated", "name": "Isolated"}}, expectedInError: `rooms are not reachable from initial room`},
		{name: "missing initial room", path: []string{"rooms"}, value: map[string]interface{}{"other": map[string]interface{}{"id": "other", "name": "Other", "npcs": []interface{}{"npc"}}}, expectedInError: `initial room "harbor_square" does not exist`},
		{name: "exit missing reverse connection", path: []string{"rooms"}, value: map[string]interface{}{"harbor_square": map[string]interface{}{"id": "harbor_square", "name": "Room", "npcs": []interface{}{"npc"}, "exits": map[string]interface{}{"north": "other"}}, "other": map[string]interface{}{"id": "other", "name": "Other"}}, expectedInError: `reverse exit "south" is missing`},
		{name: "quest ID mismatch", path: []string{"quests", "quest", "id"}, value: "different", expectedInError: `does not match quest ID`},
		{name: "NPC key mismatch", path: []string{"npcs"}, value: map[string]interface{}{"different": map[string]interface{}{"id": "npc", "name": "NPC", "quest_id": "quest", "combat_stats": map[string]interface{}{"hp": 1, "max_hp": 1, "attack": 1, "defense": 0, "speed": 1}}}, expectedInError: `NPC key "different" does not match NPC ID "npc"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var definition map[string]interface{}
			if err := json.Unmarshal([]byte(validDefinition), &definition); err != nil {
				t.Fatalf("unmarshal valid definition: %v", err)
			}
			if test.name == "multiple unique items in room" {
				definition["items"].(map[string]interface{})["item"].(map[string]interface{})["type"] = "unique"
			}
			if test.name == "unique item in multiple rooms" {
				definition["items"].(map[string]interface{})["item"].(map[string]interface{})["type"] = "unique"
			}
			if test.name == "unknown quest item" {
				definition["quests"].(map[string]interface{})["quest"].(map[string]interface{})["required_item_quantity"] = 1
			}
			current := definition
			for _, key := range test.path[:len(test.path)-1] {
				current = current[key].(map[string]interface{})
			}
			current[test.path[len(test.path)-1]] = test.value

			encoded, err := json.Marshal(definition)
			if err != nil {
				t.Fatalf("marshal invalid definition: %v", err)
			}
			_, err = world.Load(writeWorldDefinition(t, string(encoded)))
			if err == nil {
				t.Fatal("world.Load() succeeded for an invalid definition")
			}
			if !strings.Contains(err.Error(), test.expectedInError) {
				t.Fatalf("error = %q, want it to contain %q", err, test.expectedInError)
			}
		})
	}
}

func TestLoadWorldAllowsQuestChainsForSameNPC(t *testing.T) {
	definition := `{"rooms":{"harbor_square":{"id":"harbor_square",` +
		`"name":"Room","npcs":["npc"]}},"npcs":{"npc":{"id":"npc",` +
		`"name":"NPC","combat_stats":{"hp":1,"max_hp":1,"attack":1,` +
		`"defense":0,"speed":1},"quest_id":"quest.first"}},` +
		`"quests":{"quest.first":{"id":"quest.first","name":"First",` +
		`"giver_npc_id":"npc","next_quest_id":"quest.second"},` +
		`"quest.second":{"id":"quest.second","name":"Second",` +
		`"giver_npc_id":"npc"}}}`

	gameWorld, err := world.Load(writeWorldDefinition(t, definition))
	if err != nil {
		t.Fatalf("world.Load() rejected a valid NPC quest chain: %v", err)
	}

	if gameWorld == nil {
		t.Fatal("world.Load() returned a nil world without an error")
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

func TestLoadWorldJSONParsing(t *testing.T) {
	validDefinition := `{"rooms":{"harbor_square":{"id":"harbor_square","name":"Room"}}}`

	tests := []struct {
		name            string
		definition      string
		expectedInError string
	}{
		{
			name:            "unknown top-level field",
			definition:      `{"rooms":{"room":{"id":"room","name":"Room"}},"unexpected":true}`,
			expectedInError: `unknown field "unexpected"`,
		},
		{
			name:            "unknown nested field",
			definition:      `{"rooms":{"room":{"id":"room","name":"Room","unexpected":true}}}`,
			expectedInError: `unknown field "unexpected"`,
		},
		{
			name:            "duplicate top-level key",
			definition:      `{"rooms":{},"rooms":{}}`,
			expectedInError: `duplicate JSON object key "rooms"`,
		},
		{
			name:            "duplicate nested key",
			definition:      `{"rooms":{"harbor_square":{"id":"harbor_square","name":"Room","name":"Other"}}}`,
			expectedInError: `duplicate JSON object key "name"`,
		},
		{
			name:            "incompatible JSON field type",
			definition:      `{"rooms":{"harbor_square":{"id":1,"name":"Room"}}}`,
			expectedInError: "cannot unmarshal number into Go struct field",
		},
		{
			name:            "multiple JSON values",
			definition:      validDefinition + ` {}`,
			expectedInError: "multiple JSON values are not allowed",
		},
		{
			name:            "trailing non-JSON content",
			definition:      validDefinition + ` trailing`,
			expectedInError: "unexpected trailing content",
		},
		{
			name:       "trailing whitespace",
			definition: validDefinition + " \n\t",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeWorldDefinition(t, tt.definition)

			gameWorld, err := world.Load(path)
			if tt.expectedInError != "" {
				if err == nil {
					t.Fatal("world.Load() succeeded for an invalid JSON definition")
				}

				if !strings.Contains(err.Error(), tt.expectedInError) {
					t.Errorf(
						"expected error to contain %q, got %q",
						tt.expectedInError,
						err,
					)
				}

				if gameWorld != nil {
					t.Error("world.Load() returned a world together with an error")
				}
				return
			}

			if err != nil {
				t.Fatalf("world.Load() returned an error: %v", err)
			}

			if gameWorld == nil {
				t.Fatal("world.Load() returned a nil world without an error")
			}

			if got := gameWorld.RoomCount(); got != 1 {
				t.Errorf("expected 1 room, got %d", got)
			}
		})
	}
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
