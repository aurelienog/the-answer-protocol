package world

import (
	"fmt"
	"strings"
	"time"
)

// EndCombat finishes a combat, synchronizes final HP
// and distributes rewards if the players win.
func (w *World) EndCombat(combatID string) (*Combat, error) {
	if w == nil {
		return nil, fmt.Errorf(
			"%w: world is nil",
			ErrInvalidWorldState,
		)
	}

	combatID = strings.TrimSpace(combatID)
	if combatID == "" {
		return nil, fmt.Errorf(
			"%w: combat ID cannot be empty",
			ErrInvalidArgument,
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	combat, err := w.getActiveCombatLocked(combatID)
	if err != nil {
		return nil, err
	}

	defeatedNPCs, hasLivingNPC, err :=
		w.collectCombatNPCResultsLocked(combat)
	if err != nil {
		return nil, err
	}

	playersActive := combatHasActivePlayersLocked(combat)

	if hasLivingNPC && playersActive {
		return nil, fmt.Errorf(
			"%w: combat is still ongoing",
			ErrInvalidWorldState,
		)
	}

	victory := !hasLivingNPC

	if err := w.validateCombatEndLocked(
		combat,
		victory,
		defeatedNPCs,
	); err != nil {
		return nil, err
	}

	w.synchronizeCombatNPCsLocked(combat)
	w.synchronizeCombatPlayersLocked(combat, victory)

	if victory {
		// Distribute rewards to every participant who did not escape.
		if err := w.distributeCombatRewardsLocked(
			combat,
			defeatedNPCs,
		); err != nil {
			return nil, err
		}
	} else {
		w.cleanupDisconnectedCombatPlayersLocked(combat)
	}

	combat.Active = false

	return cloneCombat(combat), nil
}

// collectCombatNPCResultsLocked collects defeated NPCs
// and determines whether any NPC participants remain alive.
// The caller must hold w.mu.Lock().
func (w *World) collectCombatNPCResultsLocked(
	combat *Combat,
) ([]NPC, bool, error) {
	defeatedNPCs := make([]NPC, 0)
	hasNPC := false
	hasLivingNPC := false

	for _, participant := range combat.Participants {
		if participant.IsPlayer {
			continue
		}

		hasNPC = true

		npc, exists := w.npcs[participant.ID]
		if !exists {
			return nil, false, fmt.Errorf(
				"%w: NPC %q not found",
				ErrInvalidWorldState,
				participant.ID,
			)
		}

		if participant.Alive && participant.Stats.HP > 0 {
			hasLivingNPC = true
		} else {
			defeatedNPCs = append(defeatedNPCs, npc)
		}
	}

	if !hasNPC {
		return nil, false, fmt.Errorf(
			"%w: combat has no NPC participants",
			ErrInvalidWorldState,
		)
	}

	return defeatedNPCs, hasLivingNPC, nil
}

// validateCombatEndLocked validates the state required to finish combat.
// The caller must hold w.mu.Lock().
func (w *World) validateCombatEndLocked(
	combat *Combat,
	victory bool,
	defeatedNPCs []NPC,
) error {
	needsRespawn := !victory

	for _, participant := range combat.Participants {
		if !participant.IsPlayer || !participant.Connected {
			continue
		}

		if _, exists := w.players[participant.ID]; !exists {
			return fmt.Errorf(
				"%w: player %q not found",
				ErrInvalidWorldState,
				participant.ID,
			)
		}

		if !participant.Alive || participant.Stats.HP <= 0 {
			needsRespawn = true
		}
	}

	if needsRespawn {
		if _, exists := w.rooms[InitialRoomID]; !exists {
			return fmt.Errorf(
				"%w: initial room %q not found",
				ErrInvalidWorldState,
				InitialRoomID,
			)
		}
	}

	if victory {
		if err := w.validateCombatRewardsLocked(
			combat,
			defeatedNPCs,
		); err != nil {
			return err
		}
	}

	return nil
}

// synchronizeCombatNPCsLocked synchronizes NPC HP
// and schedules respawn when an NPC is defeated.
// The caller must hold w.mu.Lock().
func (w *World) synchronizeCombatNPCsLocked(combat *Combat) {
	for _, participant := range combat.Participants {
		if participant.IsPlayer {
			continue
		}

		npc := w.npcs[participant.ID]
		npc.CombatStats.HP = participant.Stats.HP

		if npc.CombatStats.HP <= 0 {
			respawnSeconds := npc.RespawnSeconds
			if respawnSeconds <= 0 {
				respawnSeconds = DefaultNPCRespawnSeconds
			}

			npc.RespawnAt = time.Now().Add(
				time.Duration(respawnSeconds) * time.Second,
			).Unix()
		}

		w.npcs[participant.ID] = npc
	}
}

// synchronizeCombatPlayersLocked synchronizes player HP
// and handles respawn after combat.
// The caller must hold w.mu.Lock().
func (w *World) synchronizeCombatPlayersLocked(
	combat *Combat,
	victory bool,
) {
	for _, participant := range combat.Participants {
		if !participant.IsPlayer || !participant.Connected {
			continue
		}

		player := w.players[participant.ID]

		// Dead players respawn regardless of the combat outcome.
		if !participant.Alive || participant.Stats.HP <= 0 {
			player.RoomID = InitialRoomID
			player.CombatStats.HP = player.CombatStats.MaxHP / 4
			continue
		}

		// Escaped players keep their current health and position.
		if participant.Escaped {
			player.CombatStats.HP = participant.Stats.HP
			continue
		}

		// Surviving players keep their health after victory.
		if victory {
			player.CombatStats.HP = participant.Stats.HP
			continue
		}

		// Surviving players respawn after defeat.
		player.RoomID = InitialRoomID
		player.CombatStats.HP = player.CombatStats.MaxHP / 4
	}
}

// cleanupDisconnectedCombatPlayersLocked removes disconnected players
// who did not escape after a combat defeat.
// The caller must hold w.mu.Lock().
func (w *World) cleanupDisconnectedCombatPlayersLocked(
	combat *Combat,
) {
	for _, participant := range combat.Participants {
		if !participant.IsPlayer ||
			participant.Connected ||
			participant.Escaped {
			continue
		}

		delete(w.playerInventories, participant.ID)
		delete(w.players, participant.ID)
	}
}