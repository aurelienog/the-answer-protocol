package commands

import (
	"fmt"
	"strings"
	"the-answer-protocol/internal/protocol"
)

// Dispatcher routes incoming requests to registered commands.
type Dispatcher struct {
	registry map[string]Command
}

// NewDispatcher initializes an empty command registry.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		registry: make(map[string]Command),
	}
}

// Register binds a command implementation to its name.
func (d *Dispatcher) Register(cmd Command) {
	if cmd == nil {
		panic("cannot register a nil command")
	}

	name := strings.ToLower(strings.TrimSpace(cmd.Name()))
	if name == "" {
		panic("cannot register a command with an empty name")
	}

	d.registry[name] = cmd
}

// Dispatch parses a request and executes the corresponding command.
func (d *Dispatcher) Dispatch(
	ctx *Context,
	rawInput string,
) (protocol.Message, error) {
	request, err := protocol.ParseRequest(rawInput)
	if err != nil {
		return protocol.Message{}, err
	}
	if ctx == nil {
		return protocol.Message{}, fmt.Errorf("command context is nil")
	}
	cmdName := strings.ToLower(request.Command)

	cmd, exists := d.registry[cmdName]
	if !exists {
		return protocol.ERR(
			protocol.CodeUnknownCommand,
			"UNKNOWN_COMMAND",
		), nil
	}
	ctx.RawArgs = request.Args
	ctx.RawText = request.RawArgs

	return cmd.Execute(ctx)
}
