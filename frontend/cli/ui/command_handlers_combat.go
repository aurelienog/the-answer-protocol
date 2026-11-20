package ui

import (
	"strconv"
	"strings"
)

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

func handleCombatActionResponse(model *Model, payload string) error {
	fields := parsePayloadFields(payload)

	if combatID := fields["combat"]; combatID != "" {
		model.world.Combat.ID = combatID
		model.world.Combat.Active = true
	}

	return nil
}

func handleCombatUseResponse(model *Model, payload string) error {
	fields := parsePayloadFields(payload)

	if combatID := fields["combat"]; combatID != "" {
		model.world.Combat.ID = combatID
		model.world.Combat.Active = true
	}

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

	return nil
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