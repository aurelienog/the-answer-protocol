package internal_test

import (
	"errors"
	"reflect"
	"testing"

	"the-answer-protocol/internal/commands"
	"the-answer-protocol/internal/protocol"
)

type mockCommand struct {
	name    string
	execute func(*commands.Context) (protocol.Message, error)
	calls   int
}

func (command *mockCommand) Name() string {
	return command.name
}

func (command *mockCommand) Execute(
	ctx *commands.Context,
) (protocol.Message, error) {
	command.calls++
	if command.execute != nil {
		return command.execute(ctx)
	}

	return protocol.OK("mock executed"), nil
}

func TestDispatcherRejectsNilContext(t *testing.T) {
	dispatcher := commands.NewDispatcher()
	command := &mockCommand{
		name: "LOOK",
		execute: func(*commands.Context) (protocol.Message, error) {
			return protocol.OK("executed"), nil
		},
	}
	dispatcher.Register(command)

	_, err := dispatcher.Dispatch(nil, "LOOK")
	if err == nil {
		t.Fatal("expected an error for nil context")
	}

	if command.calls != 0 {
		t.Errorf("command executed %d times for nil context, want 0", command.calls)
	}
}

func TestDispatcherReturnsProtocolErrorForUnknownCommand(t *testing.T) {
	dispatcher := commands.NewDispatcher()
	ctx := &commands.Context{}
	command := &mockCommand{
		name: "LOOK",
		execute: func(*commands.Context) (protocol.Message, error) {
			return protocol.OK("executed"), nil
		},
	}
	dispatcher.Register(command)

	message, err := dispatcher.Dispatch(ctx, "FLY")
	if err != nil {
		t.Fatalf("expected protocol error without Go error, got %v", err)
	}

	if message.Kind != protocol.KindERR {
		t.Fatalf("expected ERR message, got kind %v", message.Kind)
	}

	if message.Code != protocol.CodeUnknownCommand {
		t.Errorf(
			"expected error code %d, got %d",
			protocol.CodeUnknownCommand,
			message.Code,
		)
	}

	if message.Name != "UNKNOWN_COMMAND" {
		t.Errorf("expected error name %q, got %q", "UNKNOWN_COMMAND", message.Name)
	}

	if command.calls != 0 {
		t.Errorf(
			"registered command executed %d times for unknown input, want 0",
			command.calls,
		)
	}
}

func TestDispatcherPassesParsedArgumentsToCommand(t *testing.T) {
	dispatcher := commands.NewDispatcher()
	expectedMessage := protocol.OK("received")

	command := &mockCommand{
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

			return expectedMessage, nil
		},
	}
	dispatcher.Register(command)

	ctx := &commands.Context{}
	message, err := dispatcher.Dispatch(ctx, "chat   GLOBAL   Hello, world!")
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}

	if command.calls != 1 {
		t.Errorf("command executed %d times, want 1", command.calls)
	}

	if !reflect.DeepEqual(message, expectedMessage) {
		t.Errorf("Dispatch() message = %#v, want %#v", message, expectedMessage)
	}
}

func TestDispatcherPropagatesCommandError(t *testing.T) {
	dispatcher := commands.NewDispatcher()
	expectedError := errors.New("command execution failed")
	expectedMessage := protocol.ERR(protocol.CodeCommandFailed, "COMMAND_FAILED")

	command := &mockCommand{
		name: "FAIL",
		execute: func(_ *commands.Context) (protocol.Message, error) {
			return expectedMessage, expectedError
		},
	}
	dispatcher.Register(command)

	message, err := dispatcher.Dispatch(&commands.Context{}, "FAIL")
	if !errors.Is(err, expectedError) {
		t.Fatalf("expected command error %v, got %v", expectedError, err)
	}

	if command.calls != 1 {
		t.Errorf("command executed %d times, want 1", command.calls)
	}

	if !reflect.DeepEqual(message, expectedMessage) {
		t.Errorf("Dispatch() message = %#v, want %#v", message, expectedMessage)
	}
}

func TestDispatcherMatchesCommandNamesCaseInsensitively(t *testing.T) {
	dispatcher := commands.NewDispatcher()
	expectedMessage := protocol.OK("matched")
	command := &mockCommand{
		name: "  LoOk  ",
		execute: func(*commands.Context) (protocol.Message, error) {
			return expectedMessage, nil
		},
	}
	dispatcher.Register(command)

	message, err := dispatcher.Dispatch(&commands.Context{}, "look")
	if err != nil {
		t.Fatalf("expected command name to match case-insensitively, got %v", err)
	}

	if command.calls != 1 {
		t.Errorf("command executed %d times, want 1", command.calls)
	}

	if !reflect.DeepEqual(message, expectedMessage) {
		t.Errorf("Dispatch() message = %#v, want %#v", message, expectedMessage)
	}
}

func TestDispatcherRejectsMalformedRequest(t *testing.T) {
	dispatcher := commands.NewDispatcher()
	command := &mockCommand{
		name: "LOOK",
		execute: func(*commands.Context) (protocol.Message, error) {
			return protocol.OK("executed"), nil
		},
	}
	dispatcher.Register(command)

	_, err := dispatcher.Dispatch(&commands.Context{}, "   ")
	if err == nil {
		t.Fatal("expected an error for an empty request")
	}

	if command.calls != 0 {
		t.Errorf(
			"command executed %d times for malformed request, want 0",
			command.calls,
		)
	}
}

func TestDispatcherPreservesRawTextAndParsedArguments(t *testing.T) {
	dispatcher := commands.NewDispatcher()

	command := &mockCommand{
		name: "CHAT",
		execute: func(ctx *commands.Context) (protocol.Message, error) {
			if got, want := ctx.RawText, "GLOBAL\tHello"; got != want {
				t.Errorf("RawText = %q, want %q", got, want)
			}

			if want := []string{"GLOBAL", "Hello"}; !reflect.DeepEqual(ctx.RawArgs, want) {
				t.Errorf("RawArgs = %#v, want %#v", ctx.RawArgs, want)
			}

			return protocol.OK("received"), nil
		},
	}
	dispatcher.Register(command)

	_, err := dispatcher.Dispatch(&commands.Context{}, "CHAT\tGLOBAL\tHello")
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}

	if command.calls != 1 {
		t.Errorf("command executed %d times, want 1", command.calls)
	}
}
