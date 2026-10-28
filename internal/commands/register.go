package commands

func RegisterCommands(dispatcher *Dispatcher) {
	dispatcher.Register(NewLookCommand())
	dispatcher.Register(NewQuitCommand())
	dispatcher.Register(NewMoveCommand())
	dispatcher.Register(NewTakeCommand())
	// Register additional commands as they are implemented:
	// dispatcher.Register(NewDropCommand())
	// dispatcher.Register(NewInventoryCommand())
	// dispatcher.Register(NewAttackCommand())
	// dispatcher.Register(NewHelpCommand())
}
