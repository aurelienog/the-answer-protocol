package commands

import (
	"the-answer-protocol/internal/events"
	"the-answer-protocol/internal/world"
)

// Context provides the execution boundary environment for any game command
type Context struct {
	PlayerName string
	RawArgs    []string
	RawText    string
	World      *world.World
	Events     events.Publisher
}

