package world

import (
	"fmt"
	"strings"
)

// JoinCombat allows members of the combat's group to join at any time.
// Other players can join only after help has been requested.
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
			"%w: %q",
			ErrCombatNotFound,
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

	playerGroupID, isGroupMember := w.playerGroups[playerName]
	if isGroupMember {
		group, exists := w.groups[playerGroupID]
		if !exists {
			return nil, nil, fmt.Errorf(
				"%w: player %q references nonexistent group %q",
				ErrInvalidWorldState,
				playerName,
				playerGroupID,
			)
		}

		isGroupMember = false
		for _, member := range group.Members {
			if member == playerName {
				isGroupMember = true
				break
			}
		}

		if isGroupMember {
			isGroupMember = false
			for _, participant := range combat.Participants {
				if !participant.IsPlayer {
					continue
				}
				if participantGroupID, exists := w.playerGroups[participant.ID]; exists &&
					participantGroupID == playerGroupID {
					isGroupMember = true
					break
				}
			}
		}
	}

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
			ErrPlayerNotInCombatRoom,
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
