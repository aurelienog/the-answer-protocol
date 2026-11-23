package commands

import (
	"fmt"
	"log/slog"
	"strings"
	"the-answer-protocol/internal/protocol"
)

// Dispatcher routes incoming requests to registered commands.
type Dispatcher struct {
	registry map[string]Command
	logger   *slog.Logger
}

// NewDispatcher initializes an empty command registry.
func NewDispatcher(logger *slog.Logger) *Dispatcher {
	if logger == nil {
		panic("commands dispatcher requires a logger")
	}
	return &Dispatcher{
		registry: make(map[string]Command),
		logger:   logger,
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
		d.logger.Warn("command.parse_failed",
			slog.String("session_id", ctxSessionID(ctx)),
			slog.String("player", ctxPlayerName(ctx)),
			slog.String("error", err.Error()),
		)
		return protocol.Message{}, err
	}
	if ctx == nil {
		return protocol.Message{}, fmt.Errorf("command context is nil")
	}
	cmdName := strings.ToLower(request.Command)
	d.logger.Info("command.received",
		slog.String("session_id", ctx.SessionID),
		slog.String("player", ctx.PlayerName),
		slog.String("command", strings.ToUpper(cmdName)),
	)

	cmd, exists := d.registry[cmdName]
	if !exists {
		d.logger.Warn("command.completed",
			slog.String("session_id", ctx.SessionID),
			slog.String("player", ctx.PlayerName),
			slog.String("command", strings.ToUpper(cmdName)),
			slog.String("result", "unknown_command"),
		)
		return protocol.ERR(
			protocol.CodeUnknownCommand,
			"UNKNOWN_COMMAND",
		), nil
	}
	ctx.RawArgs = request.Args
	ctx.RawText = request.RawArgs

	message, err := cmd.Execute(ctx)
	attrs := []any{
		slog.String("session_id", ctx.SessionID),
		slog.String("player", ctx.PlayerName),
		slog.String("command", strings.ToUpper(cmdName)),
	}
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
		d.logger.Error("command.failed", attrs...)
	} else {
		d.logger.Info("command.completed", attrs...)
	}
	return message, err
}

func ctxSessionID(ctx *Context) string {
	if ctx == nil {
		return ""
	}
	return ctx.SessionID
}

func ctxPlayerName(ctx *Context) string {
	if ctx == nil {
		return ""
	}
	return ctx.PlayerName
}
