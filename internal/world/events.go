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

type ChatMessage struct {
    Scope      string
    PlayerName string
    Message    string
    Recipients []string
}

// CombatStarted represents the beginning of a combat.
type CombatStarted struct {
	CombatID    string
	RoomID      string
	Participants []string
	Recipients  []string
}

// CombatAttack represents an attack performed during combat.
type CombatAttack struct {
	CombatID    string
	AttackerID  string
	AttackerName string
	TargetID    string
	TargetName  string
	IsPlayer    bool
	Recipients  []string
}

// CombatDamage represents damage dealt to a combat participant.
type CombatDamage struct {
    CombatID    string
    AttackerID  string
    AttackerName string
    TargetID    string
    TargetName  string
    Damage      int
    RemainingHP int
    Recipients  []string
}

// CombatEnded represents the end of a combat.
type CombatEnded struct {
	CombatID   string
	Outcome    string
	Recipients []string
}

type ParticipantDied struct {
    CombatID        string
    ParticipantID   string
    ParticipantName string
    IsPlayer        bool
    KillerID        string
    Recipients      []string
}

// PlayerFled represents a player successfully escaping from combat.
type PlayerFled struct {
    CombatID   string
    PlayerID   string
    PlayerName string
    Damage     int
    RemainingHP int
    Recipients []string
}