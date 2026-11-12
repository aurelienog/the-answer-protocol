package ui

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