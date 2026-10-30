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

// UniqueItemsReturned represents unique items returned to a room
// when a player disconnects.
type UniqueItemsReturned struct {
    PlayerName string
    RoomID     string
    ItemIDs    []string
}

// GroupInvite represents a player receiving a group invitation.
type GroupInvite struct {
    GroupID   string
    GroupName string
    Inviter   string
    Invitee   string
}

// GroupJoin represents a player joining a group.
type GroupJoin struct {
    GroupID    string
    GroupName  string
    PlayerName string
    Members    []string
}

// GroupLeave represents a player leaving a group.
type GroupLeave struct {
    GroupID    string
    GroupName  string
    PlayerName string
    Members    []string
}

// GroupChat represents a message sent to a group.
type GroupChat struct {
    GroupID    string
    PlayerName string
    Message    string
    Members    []string
}