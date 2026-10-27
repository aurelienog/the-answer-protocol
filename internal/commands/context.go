package commands

import (
	"the-answer-protocol/internal/events"
	"the-answer-protocol/internal/world"
)

type Context struct {
	PlayerName string
	RawArgs    []string
	RawText    string
	World      *world.World
	Events     events.Publisher
}

// Context provides the execution boundary environment for any game command
// Here you will later inject pointers to:
// - events.Bus (to broadcast observer notifications)
