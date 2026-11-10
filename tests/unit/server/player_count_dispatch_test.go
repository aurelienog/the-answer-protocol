package server_test

import (
	"sync"
	"testing"

	"the-answer-protocol/internal/server"
	"the-answer-protocol/internal/world"
)

func TestNotificationDispatcherBroadcastsOnlyNewestPlayerCount(t *testing.T) {
	registry := server.NewRegistry()
	sessions := map[string]*server.Session{
		"Alice": newRegistryTestSession("session-alice", "Alice"),
		"Bob":   newRegistryTestSession("session-bob", "Bob"),
		"Carol": newRegistryTestSession("session-carol", "Carol"),
	}
	for _, session := range sessions {
		if err := registry.Add(session); err != nil {
			t.Fatalf("register %q: %v", session.PlayerName, err)
		}
	}

	dispatcher := server.NewNotificationDispatcher(registry)
	dispatcher.Handle(world.PlayerCountChanged{Count: 2, Revision: 2})
	dispatcher.Handle(world.PlayerCountChanged{Count: 1, Revision: 1})

	for name, session := range sessions {
		if got := len(session.Outbox); got != 1 {
			t.Errorf("%s received %d player-count notifications, want 1", name, got)
			continue
		}
		if got := <-session.Outbox; got != "EVT STATS players=2\n" {
			t.Errorf("%s player-count notification = %q, want latest count", name, got)
		}
	}
}

func TestNotificationDispatcherSerializesConcurrentPlayerCountChanges(t *testing.T) {
	registry := server.NewRegistry()
	session := &server.Session{
		ID:         "session-alice",
		PlayerName: "Alice",
		Outbox:     make(chan string, 64),
	}
	if err := registry.Add(session); err != nil {
		t.Fatalf("register Alice: %v", err)
	}

	dispatcher := server.NewNotificationDispatcher(registry)
	const revisions = 64
	var waitGroup sync.WaitGroup
	for revision := 1; revision <= revisions; revision++ {
		waitGroup.Add(1)
		go func(revision int) {
			defer waitGroup.Done()
			dispatcher.Handle(world.PlayerCountChanged{
				Count:    revision,
				Revision: uint64(revision),
			})
		}(revision)
	}
	waitGroup.Wait()

	var last string
	for len(session.Outbox) > 0 {
		last = <-session.Outbox
	}
	if want := "EVT STATS players=64\n"; last != want {
		t.Fatalf("last player-count notification = %q, want %q", last, want)
	}
}
