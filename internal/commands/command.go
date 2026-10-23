package commands

import "the-answer-protocol/internal/protocol"

// Command defines the behavioral contract that every game action must implement
type Command interface {
	Name() string
	Execute(ctx *Context) (protocol.Message, error)
}
