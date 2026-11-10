package internal_test

import (
	"fmt"
	"testing"
)

func TestHandleConnectionSendsGreeting(t *testing.T) {
	env := newConnectionTestEnv(t)

	got := env.readResponse(t)
	want := "OK hello proto=1"

	if got != want {
		t.Fatalf("greeting = %q, want %q", got, want)
	}
}

func TestHandleConnectionConnectsPlayer(t *testing.T) {
	env := newConnectionTestEnv(t)

	if got := env.readResponse(t); got != "OK hello proto=1" {
		t.Fatalf("greeting = %q", got)
	}

	env.sendRequest(t, "CONNECT Alice")

	if got := env.readResponse(t); got != "OK connected" {
		t.Fatalf("connect response = %q, want %q", got, "OK connected")
	}
}

func TestHandleConnectionConnectsUnicodeUsername(t *testing.T) {
	env := newConnectionTestEnv(t)

	_ = env.readResponse(t) // greeting
	env.sendRequest(t, "CONNECT José李")

	if got := env.readResponse(t); got != "OK connected" {
		t.Fatalf("connect response = %q, want %q", got, "OK connected")
	}

	if _, exists := env.registry.GetByName("José李"); !exists {
		t.Fatal("expected Unicode username to be present in the session registry")
	}
}

func TestHandleConnectionRequiresConnectBeforeCommands(t *testing.T) {
	env := newConnectionTestEnv(t)

	_ = env.readResponse(t) // greeting
	env.sendRequest(t, "LOOK")

	got := env.readResponse(t)
	if got != "ERR 403 CONNECT_REQUIRED" {
		t.Fatalf("response = %q, want %q", got, "ERR 403 CONNECT_REQUIRED")
	}
}

func TestHandleConnectionRejectsInvalidUsername(t *testing.T) {
	env := newConnectionTestEnv(t)

	_ = env.readResponse(t) // greeting
	env.sendRequest(t, "CONNECT invalid-name")

	got := env.readResponse(t)
	if got != "ERR 407 INVALID_USERNAME" {
		t.Fatalf("response = %q, want %q", got, "ERR 407 INVALID_USERNAME")
	}
}

func TestHandleConnectionRejectsUnknownCommand(t *testing.T) {
	env := newConnectionTestEnv(t)

	_ = env.readResponse(t) // greeting
	env.sendRequest(t, "CONNECT Alice")

	if got := env.readResponse(t); got != "OK connected" {
		t.Fatalf("connect response = %q", got)
	}

	env.sendRequest(t, "DANCE")

	got := env.readResponse(t)
	if got != "ERR 408 UNKNOWN_COMMAND" {
		t.Fatalf("response = %q, want %q", got, "ERR 408 UNKNOWN_COMMAND")
	}

	env.sendRequest(t, "WHO")
	if got := env.readResponse(t); got != "OK players=1" {
		t.Fatalf("response after recoverable error = %q, want %q", got, "OK players=1")
	}
}

func TestHandleConnectionRejectsDuplicateUsername(t *testing.T) {
	first := newConnectionTestEnv(t)

	_ = first.readResponse(t) // greeting
	first.sendRequest(t, "CONNECT Alice")

	if got := first.readResponse(t); got != "OK connected" {
		t.Fatalf("first connect response = %q", got)
	}

	// Use the same registry and world for the second connection.
	second := newSharedConnectionTestEnv(
		t,
		first.registry,
		first.dispatcher,
		first.gameWorld,
		first.eventBus,
		"second-connection",
	)

	_ = second.readResponse(t) // greeting
	second.sendRequest(t, "CONNECT Alice")

	got := second.readResponse(t)
	if got != "ERR 201 NAME_IN_USE" {
		t.Fatalf("duplicate username response = %q, want %q", got, "ERR 201 NAME_IN_USE")
	}
}

func TestHandleConnectionAllowsOnlyOneConcurrentDuplicateUsername(t *testing.T) {
	first := newConnectionTestEnv(t)
	if got := first.readResponse(t); got != "OK hello proto=1" {
		t.Fatalf("first greeting = %q", got)
	}

	second := newSharedConnectionTestEnv(
		t,
		first.registry,
		first.dispatcher,
		first.gameWorld,
		first.eventBus,
		"concurrent-connection",
	)
	if got := second.readResponse(t); got != "OK hello proto=1" {
		t.Fatalf("second greeting = %q", got)
	}

	start := make(chan struct{})
	sendResults := make(chan error, 2)
	for _, env := range []*connectionTestEnv{first, second} {
		go func(env *connectionTestEnv) {
			<-start
			_, err := fmt.Fprintln(env.client, "CONNECT Alice")
			sendResults <- err
		}(env)
	}
	close(start)

	for i := 0; i < 2; i++ {
		if err := <-sendResults; err != nil {
			t.Fatalf("send concurrent CONNECT request: %v", err)
		}
	}

	responses := []string{first.readResponse(t), second.readResponse(t)}
	successes := 0
	duplicates := 0
	for _, response := range responses {
		switch response {
		case "OK connected":
			successes++
		case "ERR 201 NAME_IN_USE":
			duplicates++
		default:
			t.Errorf("unexpected concurrent CONNECT response %q", response)
		}
	}

	if successes != 1 || duplicates != 1 {
		t.Fatalf(
			"concurrent CONNECT responses = %v, want one success and one NAME_IN_USE",
			responses,
		)
	}
}
