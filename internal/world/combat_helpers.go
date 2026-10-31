package world

// playerInActiveCombatLocked reports whether a player is already
// participating in an active combat.
// The caller must hold at least a read lock on w.mu.
func (w *World) playerInActiveCombatLocked(playerName string) bool {
	for _, combat := range w.combats {
		if combat == nil || !combat.Active {
			continue
		}

		for _, participant := range combat.Participants {
			if participant.IsPlayer &&
				participant.ID == playerName &&
				participant.Alive &&
				!participant.Escaped {
				return true
			}
		}
	}

	return false
}

// npcInActiveCombatLocked reports whether an NPC is already
// participating in an active combat.
// The caller must hold at least a read lock on w.mu.
func (w *World) npcInActiveCombatLocked(npcID string) bool {
	for _, combat := range w.combats {
		if combat == nil || !combat.Active {
			continue
		}

		for _, participant := range combat.Participants {
			if !participant.IsPlayer &&
				participant.ID == npcID &&
				participant.Alive &&
				!participant.Escaped {
				return true
			}
		}
	}

	return false
}

// cloneCombat returns an independent copy of a combat.
func cloneCombat(combat *Combat) *Combat {
	if combat == nil {
		return nil
	}

	result := *combat

	result.Participants = append(
		[]CombatParticipant(nil),
		combat.Participants...,
	)

	for i := range result.Participants {
		result.Participants[i].Effects = append(
			[]CombatEffect(nil),
			combat.Participants[i].Effects...,
		)
	}

	return &result
}

// combatHasLivingPlayersLocked reports whether the combat has
// at least one living player who has not escaped.
func combatHasLivingPlayersLocked(combat *Combat) bool {
	if combat == nil {
		return false
	}

	for _, participant := range combat.Participants {
		if participant.IsPlayer &&
			participant.Alive &&
			!participant.Escaped {
			return true
		}
	}

	return false
}

// combatHasLivingNPCsLocked reports whether the combat has
// at least one living NPC.
func combatHasLivingNPCsLocked(combat *Combat) bool {
	if combat == nil {
		return false
	}

	for _, participant := range combat.Participants {
		if !participant.IsPlayer &&
			participant.Alive &&
			!participant.Escaped {
			return true
		}
	}

	return false
}