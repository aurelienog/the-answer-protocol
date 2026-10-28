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
	return "QUIT"
}

func (c *QuitCommand) Execute(
	ctx *Context,
) (protocol.Message, error) {
	if ctx == nil {
		return protocol.ERR(protocol.CodeInternalError, "INTERNAL_ERROR"), nil
	}

	return protocol.OK("bye"), ErrQuit
}