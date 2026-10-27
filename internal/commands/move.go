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

	event, err := context.World.MovePlayer(
		context.PlayerName,
		direction,
	)
	if err != nil {
		if errors.Is(err, world.ErrNoExit) {
			return protocol.ERR(301, "NO_EXIT"), nil
		}

		return protocol.Message{}, fmt.Errorf("MOVE: %w", err)
	}

	// Publish the event only after the movement succeeds.
	if context.Events != nil {
		context.Events.Publish(event)
	}

	// The movement event will be published here in a later step.
	return protocol.OK("room=" + event.NewRoomID), nil
}
