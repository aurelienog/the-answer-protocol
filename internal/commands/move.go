package commands

import (
	"errors"
	"fmt"
	"strings"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/world"
)

type MoveCommand struct{}

func NewMoveCommand() *MoveCommand {
	return &MoveCommand{}
}

func (command *MoveCommand) Name() string {
	return "move"
}

func (command *MoveCommand) Execute(
	context *Context,
) (protocol.Message, error) {
	if context == nil {
		return protocol.Message{}, fmt.Errorf("MOVE: context is nil")
	}

	if context.World == nil {
		return protocol.Message{}, fmt.Errorf(
			"MOVE: world is not initialized",
		)
	}

	if context.PlayerName == "" {
		return protocol.Message{}, fmt.Errorf(
			"MOVE: player name is empty",
		)
	}

	if len(context.RawArgs) != 1 {
		return protocol.Message{}, fmt.Errorf(
			"MOVE: expected exactly one direction",
		)
	}

	direction := strings.ToLower(strings.TrimSpace(context.RawArgs[0]))

	if direction == "" {
		return protocol.Message{}, fmt.Errorf(
			"MOVE: direction cannot be empty",
		)
	}

	if err := context.World.MovePlayer(
		context.PlayerName,
		direction,
	); err != nil {
		if errors.Is(err, world.ErrNoExit) {
			return protocol.ERR(301, "NO_EXIT"), nil
		}

		return protocol.Message{}, fmt.Errorf("MOVE: %w", err)
	}

	roomID, err := context.World.GetPlayerRoom(context.PlayerName)
	if err != nil {
		return protocol.Message{}, fmt.Errorf("MOVE: %w", err)
	}

	return protocol.OK("room=" + roomID), nil
}