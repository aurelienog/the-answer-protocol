package commands

func RegisterCommands(dispatcher *Dispatcher) {
    dispatcher.Register(NewLookCommand())
	dispatcher.Register(NewQuitCommand())

    // Register additional commands as they are implemented:
    // dispatcher.Register(NewMoveCommand())
    // dispatcher.Register(NewTakeCommand())
    // dispatcher.Register(NewDropCommand())
    // dispatcher.Register(NewInventoryCommand())
    // dispatcher.Register(NewAttackCommand())
    // dispatcher.Register(NewHelpCommand())
}