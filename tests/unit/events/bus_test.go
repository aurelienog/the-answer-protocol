package events_test

import (
	"sync"
	"testing"

	"the-answer-protocol/internal/events"
)

func TestBusPublishesToSubscribersInRegistrationOrder(t *testing.T) {
	bus := events.NewBus()
	var got []string
	bus.Subscribe(func(event any) {
		got = append(got, "first:"+event.(string))
	})
	bus.Subscribe(func(event any) {
		got = append(got, "second:"+event.(string))
	})

	bus.Publish("hello")

	want := []string{"first:hello", "second:hello"}
	if len(got) != len(want) {
		t.Fatalf("published events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("published events = %v, want %v", got, want)
		}
	}
}

func TestBusIgnoresNilBusAndNilHandler(t *testing.T) {
	var nilBus *events.Bus
	nilBus.Subscribe(func(any) {
		t.Fatal("nil bus invoked a handler")
	})
	nilBus.Publish("ignored")

	bus := events.NewBus()
	bus.Subscribe(nil)
	called := false
	bus.Subscribe(func(any) {
		called = true
	})
	bus.Publish("event")
	if !called {
		t.Fatal("non-nil handler was not called after nil subscription")
	}
}

func TestBusSupportsConcurrentSubscribeAndPublish(t *testing.T) {
	bus := events.NewBus()
	var calls sync.WaitGroup
	for i := 0; i < 8; i++ {
		calls.Add(1)
		go func() {
			defer calls.Done()
			bus.Subscribe(func(any) {})
			bus.Publish("event")
		}()
	}
	calls.Wait()
}
