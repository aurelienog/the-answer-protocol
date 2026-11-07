package internal_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"the-answer-protocol/internal/world"
)

func TestHandleConnectionReturnsUniqueItemsToRoomOnDisconnect(t *testing.T) {
	gameWorld := loadWorldWithUniqueHarborItem(t)
	bob := newConnectionTestEnvWithWorld(t, gameWorld)
	connectTestPlayer(t, bob, "Bob")

	alice := newSharedConnectionTestEnv(
		t,
		bob.registry,
		bob.dispatcher,
		bob.gameWorld,
		bob.eventBus,
		"unique-item-alice",
	)
	connectTestPlayer(t, alice, "Alice")
	if got := bob.readResponse(t); got != "EVT ROOM PRESENCE ENTER Alice" {
		t.Fatalf("Bob's Alice-enter event = %q", got)
	}

	alice.sendRequest(t, "TAKE Purse of Gold Coins")
	if got := alice.readResponse(t); got != "OK taken=item.gold_coins" {
		t.Fatalf("take unique item response = %q", got)
	}
	inventory, err := gameWorld.GetInventory("Alice")
	if err != nil {
		t.Fatalf("GetInventory(Alice) returned an error: %v", err)
	}
	if len(inventory) != 1 || inventory[0].ID != "item.gold_coins" {
		t.Fatalf("Alice's inventory before disconnect = %+v, want the unique gold coins", inventory)
	}

	if err := alice.client.Close(); err != nil {
		t.Fatalf("close Alice's connection: %v", err)
	}
	returnedEvent := bob.readResponse(t)
	if !strings.HasPrefix(returnedEvent, "EVT UNIQUE_ITEMS_RETURNED instance:") {
		t.Fatalf("unique-item return event = %q", returnedEvent)
	}
	if got := bob.readResponse(t); got != "EVT ROOM PRESENCE LEAVE Alice" {
		t.Fatalf("Bob's Alice-leave event = %q", got)
	}

	deadline := time.Now().Add(connectionTestTimeout)
	for time.Now().Before(deadline) {
		if _, ok := bob.registry.GetByName("Alice"); !ok {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if _, ok := bob.registry.GetByName("Alice"); ok {
		t.Fatal("Alice's session remains registered after disconnect")
	}
	assertPlayerRemoved(t, bob, "Alice")

	room, err := gameWorld.GetRoom("harbor_square")
	if err != nil {
		t.Fatalf("GetRoom(harbor_square) returned an error: %v", err)
	}
	if got := room.Items["item.gold_coins"]; got != 1 {
		t.Fatalf("gold coins in harbor square after disconnect = %d, want 1", got)
	}
	items, err := gameWorld.GetRoomItems("harbor_square")
	if err != nil {
		t.Fatalf("GetRoomItems(harbor_square) returned an error: %v", err)
	}
	returnedCount := 0
	for _, item := range items {
		if item.ID == "item.gold_coins" {
			returnedCount++
		}
	}
	if returnedCount != 1 {
		t.Fatalf("returned gold coin instances = %d, want exactly 1", returnedCount)
	}
}

func loadWorldWithUniqueHarborItem(t *testing.T) *world.World {
	t.Helper()

	data, err := os.ReadFile("../../data/world.json")
	if err != nil {
		t.Fatalf("read world fixture: %v", err)
	}

	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode world fixture: %v", err)
	}
	var rooms map[string]json.RawMessage
	if err := json.Unmarshal(document["rooms"], &rooms); err != nil {
		t.Fatalf("decode world rooms: %v", err)
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

	harborItems, err := json.Marshal(items)
	if err != nil {
		t.Fatalf("encode harbor items: %v", err)
	}
	harbor["items"] = harborItems
	harborData, err := json.Marshal(harbor)
	if err != nil {
		t.Fatalf("encode harbor square: %v", err)
	}
	rooms["harbor_square"] = harborData
	roomsData, err := json.Marshal(rooms)
	if err != nil {
		t.Fatalf("encode world rooms: %v", err)
	}
	document["rooms"] = roomsData
	worldData, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode world fixture: %v", err)
	}

	path := filepath.Join(t.TempDir(), "world.json")
	if err := os.WriteFile(path, worldData, 0o600); err != nil {
		t.Fatalf("write temporary world fixture: %v", err)
	}
	gameWorld, err := world.Load(path)
	if err != nil {
		t.Fatalf("load temporary world fixture: %v", err)
	}
	return gameWorld
}
