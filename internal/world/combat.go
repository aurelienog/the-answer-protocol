package world

type CombatStats struct {
    HP      int `json:"hp"`
    MaxHP   int `json:"max_hp"`
    Attack  int `json:"attack"`
    Defense int `json:"defense"`
    Speed   int `json:"speed"`
}

type Combat struct {
    ID            string              `json:"id"`
    RoomID        string              `json:"room_id"`
    GroupID       string              `json:"group_id,omitempty"`
    Participants  []CombatParticipant `json:"participants"`
    TurnIndex     int                 `json:"turn_index"`
    Round         int                 `json:"round"`
    Active        bool                `json:"active"`
    HelpRequested bool                `json:"help_requested"`
}

type CombatParticipant struct {
    ID          string         `json:"id"`
    Name        string         `json:"name"`
    IsPlayer    bool           `json:"is_player"`
    Connected   bool           `json:"connected"`
    Stats       CombatStats    `json:"stats"`
    Effects     []CombatEffect `json:"effects,omitempty"`
    SpeedPoints int            `json:"speed_points"`
    Alive       bool           `json:"alive"`
    Escaped     bool           `json:"escaped"`
}

type CombatEffect struct {
    ID              string `json:"id"`
    Name            string `json:"name"`
    SpeedModifier   int    `json:"speed_modifier"`
    RemainingRounds int    `json:"remaining_rounds"`
}