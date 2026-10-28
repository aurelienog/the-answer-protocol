package internal_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/protocol"
)

type mockCommand struct {
	name    string
	execute func(*commands.Context) (protocol.Message, error)
}

func (command *mockCommand) Name() string {
	return command.name
}

func (command *mockCommand) Execute(
	ctx *commands.Context,
) (protocol.Message, error) {
	if command.execute != nil {
		return command.execute(ctx)
	}

	return protocol.OK("mock executed"), nil
}

func TestDispatcherRejectsNilContext(t *testing.T) {
	dispatcher := commands.NewDispatcher()
	dispatcher.Register(&mockCommand{name: "LOOK"})

	_, err := dispatcher.Dispatch(nil, "LOOK")
	if err == nil {
		t.Fatal("expected an error for nil context")
	}
}

func TestDispatcherRejectsUnknownCommand(t *testing.T) {
	dispatcher := commands.NewDispatcher()
	ctx := &commands.Context{}

	_, err := dispatcher.Dispatch(ctx, "FLY")
	if !errors.Is(err, commands.ErrUnknownCommand) {
		t.Fatalf(
			"expected ErrUnknownCommand, got %v",
			err,
		)
	}
}

func TestDispatcherPassesParsedArgumentsToCommand(t *testing.T) {
	dispatcher := commands.NewDispatcher()

	dispatcher.Register(&mockCommand{
		name: "CHAT",
		execute: func(ctx *commands.Context) (protocol.Message, error) {
			expectedArgs := []string{"GLOBAL", "Hello,", "world!"}

			if !reflect.DeepEqual(ctx.RawArgs, expectedArgs) {
				t.Errorf(
					"RawArgs = %#v, want %#v",
					ctx.RawArgs,
					expectedArgs,
				)
			}

			if ctx.RawText != "GLOBAL   Hello, world!" {
				t.Errorf(
					"RawText = %q, want %q",
					ctx.RawText,
					"GLOBAL   Hello, world!",
				)
			}

			return protocol.OK("received"), nil
		},
	})

	ctx := &commands.Context{}
	_, err := dispatcher.Dispatch(ctx, "chat   GLOBAL   Hello, world!")
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
}

func TestDispatcherPropagatesCommandError(t *testing.T) {
	dispatcher := commands.NewDispatcher()
	expectedError := errors.New("command execution failed")

	dispatcher.Register(&mockCommand{
		name: "FAIL",
		execute: func(_ *commands.Context) (protocol.Message, error) {
			return protocol.Message{}, expectedError
		},
	})

	_, err := dispatcher.Dispatch(&commands.Context{}, "FAIL")
	if !errors.Is(err, expectedError) {
		t.Fatalf("expected command error %v, got %v", expectedError, err)
	}
}

func TestDispatcherMatchesCommandNamesCaseInsensitively(t *testing.T) {
	dispatcher := commands.NewDispatcher()
	dispatcher.Register(&mockCommand{name: "  LoOk  "})

	_, err := dispatcher.Dispatch(&commands.Context{}, "look")
	if err != nil {
		t.Fatalf("expected command name to match case-insensitively, got %v", err)
	}
}

func TestDispatcherRejectsMalformedRequest(t *testing.T) {
	dispatcher := commands.NewDispatcher()

	_, err := dispatcher.Dispatch(&commands.Context{}, "   ")
	if err == nil {
		t.Fatal("expected an error for an empty request")
	}

	if errors.Is(err, commands.ErrUnknownCommand) {
		t.Fatal("empty request should not be classified as an unknown command")
	}
}

func TestDispatcherDoesNotTrimRawTextInternally(t *testing.T) {
	dispatcher := commands.NewDispatcher()

	dispatcher.Register(&mockCommand{
		name: "CHAT",
		execute: func(ctx *commands.Context) (protocol.Message, error) {
			if strings.Contains(ctx.RawText, "\t") {
				t.Errorf("unexpected tab in RawText: %q", ctx.RawText)
			}

			if ctx.RawText != "GLOBAL Hello" {
				t.Errorf("RawText = %q, want %q", ctx.RawText, "GLOBAL Hello")
			}

			return protocol.OK("received"), nil
		},
	})

	_, err := dispatcher.Dispatch(&commands.Context{}, "CHAT GLOBAL Hello")
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
}
