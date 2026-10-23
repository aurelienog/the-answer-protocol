package commands

type Context struct {
    PlayerName string
    RawArgs    []string
	RawText    string
}

// Context provides the execution boundary environment for any game command
	// Here you will later inject pointers to:
	// - game.World (to modify room states)
	// - events.Bus (to broadcast observer notifications)
