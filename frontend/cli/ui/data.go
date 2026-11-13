package ui

import "the-answer-protocol/internal/world"

type LookResponse struct {
    Room    world.Room         `json:"room"`
    Players []string           `json:"players"`
    Items   []world.Item       `json:"items"`
    NPCs    []world.NPCSummary `json:"npcs"`
}

type Quest struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
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