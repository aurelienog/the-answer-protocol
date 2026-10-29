package world

// PlayerMoved represents a successfully completed player movement.
// The player lists contain the occupants of each room immediately
// before the movement, including the player who moved.
type PlayerMoved struct {
	PlayerName          string
	PreviousRoomID      string
	NewRoomID           string
	PreviousRoomPlayers []string
	NewRoomPlayers      []string
}

// PlayerEntered represents a player entering the world.
type PlayerEntered struct {
	PlayerName  string
	RoomID      string
	RoomPlayers []string
}

// PlayerLeft represents a player leaving the world.
type PlayerLeft struct {
	PlayerName  string
	RoomID      string
	RoomPlayers []string
}

// ItemDropped represents a common item successfully dropped in a room.
type ItemDropped struct {
	PlayerName string
	ItemID     string
	RoomID     string
	RoomPlayers []string
}