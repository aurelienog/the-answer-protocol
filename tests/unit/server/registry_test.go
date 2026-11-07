package server_test

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"the-answer-protocol/internal/server"
)

func TestRegistryNamesAreUniqueIgnoringCase(t *testing.T) {
	registry := server.NewRegistry()

	first := newRegistryTestSession("session-1", "Alice")
	if err := registry.Add(first); err != nil {
		t.Fatalf("Add(first) returned an error: %v", err)
	}

	if got, ok := registry.GetByName("  aLiCe "); !ok || got != first {
		t.Fatalf("GetByName() = (%p, %v), want (%p, true)", got, ok, first)
	}

	err := registry.Add(newRegistryTestSession("session-2", "ALICE"))
	if !errors.Is(err, server.ErrNameInUse) {
		t.Fatalf("Add(duplicate name) error = %v, want %v", err, server.ErrNameInUse)
	}

	if _, ok := registry.Get("session-2"); ok {
		t.Fatal("duplicate session was registered")
	}
}

func TestRegistryRemoveReleasesPlayerName(t *testing.T) {
	registry := server.NewRegistry()
	first := newRegistryTestSession("session-1", "Alice")

	if err := registry.Add(first); err != nil {
		t.Fatalf("Add(first) returned an error: %v", err)
	}

	registry.Remove(first.ID)

	if _, ok := registry.Get(first.ID); ok {
		t.Fatal("removed session is still registered")
	}

	if _, ok := registry.GetByName("alice"); ok {
		t.Fatal("removed player name is still registered")
	}

	reconnected := newRegistryTestSession("session-2", "ALICE")
	if err := registry.Add(reconnected); err != nil {
		t.Fatalf("Add(reconnected player) returned an error: %v", err)
	}
}

func TestRegistryConcurrentAddsAllowOnlyOneMatchingName(t *testing.T) {
	const contenders = 32

	registry := server.NewRegistry()
	start := make(chan struct{})
	var waitGroup sync.WaitGroup
	var successes int32
	var duplicates int32
	errorsFound := make(chan error, contenders)

	for i := 0; i < contenders; i++ {
		waitGroup.Add(1)
		go func(i int) {
			defer waitGroup.Done()
			<-start

			name := "Alice"
			if i%2 == 1 {
				name = "aLiCe"
			}

			err := registry.Add(newRegistryTestSession(
				fmt.Sprintf("session-%d", i),
				name,
			))
			switch {
			case err == nil:
				atomic.AddInt32(&successes, 1)
			case errors.Is(err, server.ErrNameInUse):
				atomic.AddInt32(&duplicates, 1)
			default:
				errorsFound <- fmt.Errorf("Add(session-%d) returned unexpected error: %w", i, err)
			}
		}(i)
	}

	close(start)
	waitGroup.Wait()
	close(errorsFound)

	for err := range errorsFound {
		t.Error(err)
	}

	if got := atomic.LoadInt32(&successes); got != 1 {
		t.Errorf("successful Add calls = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&duplicates); got != contenders-1 {
		t.Errorf("duplicate-name errors = %d, want %d", got, contenders-1)
	}

	if _, ok := registry.GetByName("ALICE"); !ok {
		t.Error("winning session is not retrievable by name")
	}
}

func newRegistryTestSession(id, playerName string) *server.Session {
	return &server.Session{
		ID:         id,
		PlayerName: playerName,
		Outbox:     make(chan string, 1),
	}
}
