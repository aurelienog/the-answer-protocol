package ui

import (
	"encoding/json"
	"strconv"
	"strings"
)

type attackResponsePayload struct {
	CombatID   string `json:"combat_id"`
	AttackerHP int    `json:"attacker_hp"`
	TargetHP   int    `json:"target_hp"`
	Damage     int    `json:"damage"`
	Status     string `json:"status"`
}

func parsePayloadFields(payload string) map[string]string {
	fields := make(map[string]string)

	for _, token := range strings.Fields(payload) {
		key, value, found := strings.Cut(token, "=")
		if found {
			fields[key] = value
		}
	}

	return fields
}

func handleAttackResponse(model *Model, payload string) error {
	var data attackResponsePayload
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return handleCombatActionResponse(model, payload)
	}

	if data.CombatID != "" {
		model.world.Combat.ID = data.CombatID
		model.world.Combat.Active = data.Status != "ended"
	}

	// Only apply response HPs if EVT COMBAT STARTED hasn't already populated the combat state.
	if len(model.world.Combat.Participants) == 0 {
		model.setCombatPlayerHealth(data.AttackerHP)
	}

	return nil
}

func handleCombatActionResponse(model *Model, payload string) error {
	fields := parsePayloadFields(payload)

	if combatID := fields["combat"]; combatID != "" {
		model.world.Combat.ID = combatID
		model.world.Combat.Active = true
	}

	return nil
}

func (model Model) isPlayerCombatTurn() bool {
	if !model.world.Combat.Active || model.world.Combat.CurrentTurn == "" {
		return false
	}

	for _, participant := range model.world.Combat.Participants {
		if !participant.IsPlayer ||
			(!strings.EqualFold(participant.Name, model.player.Username) &&
				!strings.EqualFold(participant.ID, model.player.Username)) {
			continue
		}

		return strings.EqualFold(model.world.Combat.CurrentTurn, participant.Name) ||
			strings.EqualFold(model.world.Combat.CurrentTurn, participant.ID)
	}

	return strings.EqualFold(model.world.Combat.CurrentTurn, model.player.Username)
}

func handleCombatUseResponse(model *Model, payload string) error {
	fields := parsePayloadFields(payload)

	if combatID := fields["combat"]; combatID != "" {
		model.world.Combat.ID = combatID
		model.world.Combat.Active = true
	}

	applyItemUseChanges(model, fields)

	return nil
}

func applyItemUseChanges(model *Model, fields map[string]string) {
	if changesText, exists := fields["changes"]; exists {
		for _, change := range strings.Split(changesText, ",") {
			statName, deltaText, found := strings.Cut(change, "=")
			if !found || statName != "hp" {
				continue
			}

			delta, err := strconv.Atoi(deltaText)
			if err == nil {
				model.applyCombatPlayerHealthDelta(delta)
			}
		}
	}
}

func handleCombatFleeResponse(model *Model, payload string) error {
	fields := parsePayloadFields(payload)

	if remainingHealthText, exists := fields["hp"]; exists {
		remainingHealth, err := strconv.Atoi(remainingHealthText)
		if err == nil {
			model.setCombatPlayerHealth(remainingHealth)
		}
	}

	for index := range model.world.Combat.Participants {
		participant := &model.world.Combat.Participants[index]
		if participant.IsPlayer && strings.EqualFold(participant.Name, model.player.Username) {
			participant.Escaped = true
			break
		}
	}

	return nil
}

func (model *Model) setCombatPlayerHealth(newHealth int) {
	if newHealth < 0 {
		newHealth = 0
	}
	if model.player.Status.MaxHP > 0 && newHealth > model.player.Status.MaxHP {
		newHealth = model.player.Status.MaxHP
	}

	model.player.Status.HP = newHealth

	for index := range model.world.Combat.Participants {
		participant := &model.world.Combat.Participants[index]
		if participant.IsPlayer && strings.EqualFold(participant.Name, model.player.Username) {
			participant.HP = newHealth
			participant.Alive = newHealth > 0
			break
		}
	}
}

func (model *Model) applyCombatPlayerHealthDelta(delta int) {
	model.setCombatPlayerHealth(model.player.Status.HP + delta)
}
