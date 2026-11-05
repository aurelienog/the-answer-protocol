package world

import (
	"fmt"
	"strings"
)

// RequestCombatHelp allows a player to request assistance
// from other players in the combat room.
func (w *World) RequestCombatHelp(
	playerName string,
) (*Combat, *CombatHelpRequested, error) {
	if w == nil {
		return nil, nil, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	playerName = strings.TrimSpace(playerName)
	if playerName == "" {
		return nil, nil, ErrInvalidArgument
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.players[playerName]; !exists {
		return nil, nil, fmt.Errorf(
			"%w: %q",
			ErrPlayerNotFound,
			playerName,
		)
	}

	combat := w.findActiveCombatForPlayerLocked(playerName)
	if combat == nil {
		return nil, nil, ErrPlayerNotInActiveCombat
	}

	combat.HelpRequested = true

	event := &CombatHelpRequested{
		CombatID:   combat.ID,
		PlayerName: playerName,
		RoomID:     combat.RoomID,
		Recipients: w.getPlayersInRoomLocked(combat.RoomID),
	}

	return cloneCombat(combat), event, nil
}