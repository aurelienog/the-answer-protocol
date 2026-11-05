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
) (*Combat, *CombatPlayerJoined, error) {
	if w == nil {
		return nil, nil, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
	}

	playerName = strings.TrimSpace(playerName)
	combatID = strings.TrimSpace(combatID)

	if playerName == "" || combatID == "" {
		return nil, nil, fmt.Errorf(
			"%w: player name and combat ID cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	player, exists := w.players[playerName]
	if !exists {
		return nil, nil, fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
	}

	combat, exists := w.combats[combatID]
	if !exists || combat == nil {
		return nil, nil, fmt.Errorf(
			"%w: combat %q does not exist",
			ErrInvalidWorldState,
			combatID,
		)
	}

	if !combat.Active {
		return nil, nil, fmt.Errorf(
			"%w: combat %q is not active",
			ErrInvalidWorldState,
			combatID,
		)
	}

	// Prevent players from rejoining a combat they previously participated in.
	if w.playerAlreadyParticipatedInCombatLocked(playerName, combat) {
		return nil, nil, fmt.Errorf(
			"%w: player %q has already participated in combat %q",
			ErrPlayerAlreadyInCombat,
			playerName,
			combatID,
		)
	}

	// Members of the initiating group can always join.
    // Other players require a help request.
    playerGroupID, inGroup := w.playerGroups[playerName]
    isGroupMember := combat.GroupID != "" &&
        inGroup &&
        playerGroupID == combat.GroupID

    if !isGroupMember && !combat.HelpRequested {
		return nil, nil, fmt.Errorf(
			"%w: combat %q",
			ErrCombatHelpNotRequested,
			combatID,
		)
	}

	if player.RoomID != combat.RoomID {
		return nil, nil, fmt.Errorf(
			"%w: player %q is not in combat room %q",
			ErrInvalidWorldState,
			playerName,
			combat.RoomID,
		)
	}

	if player.CombatStats.HP <= 0 {
		return nil, nil, fmt.Errorf(
			"%w: player %q is defeated",
			ErrInvalidWorldState,
			playerName,
		)
	}

	if w.playerInActiveCombatLocked(playerName) {
		return nil, nil, fmt.Errorf(
			"%w: player %q is already in an active combat",
			ErrInvalidWorldState,
			playerName,
		)
	}

	// Calculate effective stats, including equipment bonuses.
    effectiveStats, err := w.calculatePlayerCombatStatsLocked(playerName)
    if err != nil {
        return nil, nil, err
    }

    combat.Participants = append(
        combat.Participants,
        CombatParticipant{
            ID:          playerName,
            Name:        playerName,
            IsPlayer:    true,
            Connected:   true,
            Stats:       effectiveStats,
            SpeedPoints: 0,
            Alive:       true,
        },
    )

	event := &CombatPlayerJoined{
        CombatID:   combat.ID,
        PlayerName: playerName,
        Recipients: w.getCombatRecipientsLocked(combat),
    }

    return cloneCombat(combat), event, nil
}

// playerAlreadyParticipatedInCombatLocked reports whether a player
// has already participated in the specified combat.
// The caller must hold at least a read lock on w.mu.
func (w *World) playerAlreadyParticipatedInCombatLocked(
	playerName string,
	combat *Combat,
) bool {
	if combat == nil {
		return false
	}

	for _, participant := range combat.Participants {
		if participant.IsPlayer && participant.ID == playerName {
			return true
		}
	}

	return false
}
