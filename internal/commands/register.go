package commands

func RegisterCommands(dispatcher *Dispatcher) {
	dispatcher.Register(NewLookCommand())
	dispatcher.Register(NewQuitCommand())
	dispatcher.Register(NewMoveCommand())
	dispatcher.Register(NewTakeCommand())
	dispatcher.Register(NewDropCommand())
	dispatcher.Register(NewInventoryCommand())
	dispatcher.Register(NewTalkCommand())
	dispatcher.Register(NewQuestCommand())
	dispatcher.Register(NewQuestsCommand())
	dispatcher.Register(NewWhoCommand())
	dispatcher.Register(NewGroupCommand())
	dispatcher.Register(NewChatCommand())
	dispatcher.Register(NewStatusCommand())
	// dispatcher.Register(NewAttackCommand())
	// dispatcher.Register(NewHelpCommand())
}