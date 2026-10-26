package internal_test

import (
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

	if got := gameWorld.RoomCount(); got != 8 {
		t.Fatalf("expected 8 rooms, got %d", got)
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
}