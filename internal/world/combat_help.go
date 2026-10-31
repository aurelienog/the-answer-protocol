package world

import (
	"fmt"
	"strings"
)

// RequestCombatHelp allows any living player participant
// to request help for an active combat.
func (w *World) RequestCombatHelp(
	playerName string,
	combatID string,
) (*Combat, error) {
	if w == nil {
		return nil, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	playerName = strings.TrimSpace(playerName)
	combatID = strings.TrimSpace(combatID)

	if playerName == "" || combatID == "" {
		return nil, fmt.Errorf(
			"%w: player name and combat ID cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.players[playerName]; !exists {
		return nil, fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
	}

	combat, exists := w.combats[combatID]
	if !exists || combat == nil {
		return nil, fmt.Errorf(
			"%w: combat %q does not exist",
			ErrInvalidWorldState,
			combatID,
		)
	}

	if !combat.Active {
		return nil, fmt.Errorf(
			"%w: combat %q is not active",
			ErrInvalidWorldState,
			combatID,
		)
	}

	isParticipant := false

	for _, participant := range combat.Participants {
		if participant.IsPlayer &&
			participant.ID == playerName &&
			participant.Alive &&
			!participant.Escaped {
			isParticipant = true
			break
		}
	}

	if !isParticipant {
		return nil, fmt.Errorf(
			"%w: player %q is not an active combat participant",
			ErrInvalidWorldState,
			playerName,
		)
	}

	combat.HelpRequested = true

	return cloneCombat(combat), nil
}
