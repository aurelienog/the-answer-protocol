package world

import (
	"fmt"
	"strings"
)

// JoinCombat allows an external player to join a combat
// when help has been requested.
func (w *World) JoinCombat(
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

	player, exists := w.players[playerName]
	if !exists {
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

	if !combat.HelpRequested {
		return nil, fmt.Errorf(
			"%w: combat %q has not requested help",
			ErrInvalidWorldState,
			combatID,
		)
	}

	if player.RoomID != combat.RoomID {
		return nil, fmt.Errorf(
			"%w: player %q is not in combat room %q",
			ErrInvalidWorldState,
			playerName,
			combat.RoomID,
		)
	}

	if player.CombatStats.HP <= 0 {
		return nil, fmt.Errorf(
			"%w: player %q is defeated",
			ErrInvalidWorldState,
			playerName,
		)
	}

	if w.playerInActiveCombatLocked(playerName) {
		return nil, fmt.Errorf(
			"%w: player %q is already in an active combat",
			ErrInvalidWorldState,
			playerName,
		)
	}

	combat.Participants = append(
		combat.Participants,
		CombatParticipant{
			ID:       playerName,
			Name:     playerName,
			IsPlayer: true,
			Stats:    player.CombatStats,
			Alive:    true,
		},
	)

	return cloneCombat(combat), nil
}
