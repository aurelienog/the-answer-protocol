package ui


type Quest struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
    GiverNPCName string `json:"giver_npc_name,omitempty"`
    Status      string `json:"status"`
    Progress    string `json:"progress,omitempty"`
}

type Group struct {
    ID          string          `json:"ID"`
    Name        string          `json:"Name"`
    Members     []string        `json:"Members"`
    Invitations map[string]bool `json:"Invitations"`
}

type PlayerStatus struct {
    HP     int    `json:"hp"`
    MaxHP  int    `json:"max_hp"`
    Status string `json:"status"`
}

type ChatMessage struct {
    Scope      string
    PlayerName string
    Message    string
}

type LookResponse struct {
    Room    Room         `json:"room"`
    Players []string     `json:"players"`
    Items   []Item       `json:"items"`
    NPCs    []NPCSummary `json:"npcs"`
}

type QuestSummary struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
    Progress    string `json:"progress,omitempty"`
}

type NPCConversation struct {
    NPC         NPCSummary   `json:"npc"`
    Dialogue    []string     `json:"dialogue"`
    Quest       *QuestSummary `json:"quest,omitempty"`
    QuestStatus string       `json:"quest_status,omitempty"`
}


type NPCSummary struct {
    ID      string `json:"id"`
    Name    string `json:"name"`
    Role    string `json:"role"`
    Hostile bool   `json:"hostile"`
}

type CombatParticipantState struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    IsPlayer  bool   `json:"is_player"`
    HP        int    `json:"hp"`
    MaxHP     int    `json:"max_hp"`
    Alive     bool   `json:"alive"`
    Connected bool   `json:"-"`
    Escaped   bool   `json:"-"`
}

type CombatState struct {
    ID               string
    Active           bool
    Round            int
    Participants     []CombatParticipantState
    CurrentTurn      string
    AvailableActions int
    Log              []string
}