package commands

func RegisterCommands(dispatcher *Dispatcher) {
	dispatcher.Register(NewLookCommand())
	dispatcher.Register(NewQuitCommand())
	dispatcher.Register(NewMoveCommand())
	dispatcher.Register(NewTakeCommand())
	dispatcher.Register(NewDropCommand())
	dispatcher.Register(NewInventoryCommand())
	// Register additional commands as they are implemented:
	// dispatcher.Register(NewAttackCommand())
	// dispatcher.Register(NewHelpCommand())
}
