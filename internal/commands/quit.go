package commands

import (
	"errors"

	"the-answer-protocol/internal/protocol"
)

var ErrQuit = errors.New("player requested disconnect")

type QuitCommand struct{}

func NewQuitCommand() *QuitCommand {
	return &QuitCommand{}
}

func (c *QuitCommand) Name() string {
	return "quit"
}

func (c *QuitCommand) Execute(
	ctx *Context,
) (protocol.Message, error) {
	return protocol.OK("bye"), ErrQuit
}