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
	TurnsInitialized bool `json:"turns_initialized"`
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
	TargetID        string `json:"target_id"`

	Duration        int `json:"duration"`
	RemainingTurns  int `json:"remaining_turns"`

	Stat            string `json:"stat"`
	Modifier        int `json:"modifier"`
	ModifierPercent int `json:"modifier_percent"`
}

const (
	CombatStatAttack  = "attack"
	CombatStatDefense = "defense"
	CombatStatSpeed   = "speed"
	CombatStatHP      = "hp"
)

const (
    DefaultPlayerHP      = 100
    DefaultPlayerAttack  = 10
    DefaultPlayerDefense = 5
    DefaultPlayerSpeed   = 10
	DefaultDefenseBonusPercent = 50

    CombatActionCost = 10
    MinimumDamage    = 1
	FleeHealthPenaltyPercent = 15
)

func calculateDamage(attack, defense int) int {
	damage := attack - defense

	if damage < MinimumDamage {
		return MinimumDamage
	}

	return damage
}
