package world

import "fmt"

// distributeCombatRewardsLocked distributes the rewards of all defeated NPCs
// to participating players who did not escape.
// The caller must hold w.mu.Lock().
func (w *World) distributeCombatRewardsLocked(
	combat *Combat,
	defeatedNPCs []NPC,
) error {
	if combat == nil {
		return fmt.Errorf("%w: nil combat", ErrInvalidArgument)
	}

	// Validate every reward item before creating instances.
	for _, npc := range defeatedNPCs {
		for _, itemID := range npc.Rewards {
			if _, exists := w.items[itemID]; !exists {
				return fmt.Errorf("%w: %q", ErrItemNotFound, itemID)
			}
		}
	}

	// Prepare rewards without modifying player inventories.
	pendingRewards := make(map[string][]string)
	previousCounter := w.nextItemInstanceID
	createdInstances := make([]string, 0)

	for _, participant := range combat.Participants {
		if !participant.IsPlayer || participant.Escaped {
			continue
		}

		for _, npc := range defeatedNPCs {
			for _, itemID := range npc.Rewards {
				instanceID, err := w.createItemInstanceLocked(itemID)
				if err != nil {
					// Roll back all instances created during preparation.
					for _, createdID := range createdInstances {
						delete(w.itemInstances, createdID)
					}
					w.nextItemInstanceID = previousCounter
					return err
				}

				createdInstances = append(createdInstances, instanceID)
				pendingRewards[participant.ID] = append(
					pendingRewards[participant.ID],
					instanceID,
				)
			}
		}
	}

	// Commit all rewards only after successful preparation.
	
	if len(pendingRewards) > 0 && w.playerInventories == nil {
		w.playerInventories = make(map[string][]string)
	}
	
	
	for playerName, instanceIDs := range pendingRewards {
		w.playerInventories[playerName] = append(
			w.playerInventories[playerName],
			instanceIDs...,
		)
	}

	return nil
}

// validateCombatRewardsLocked validates reward item references
// The caller must hold w.mu.Lock().
func (w *World) validateCombatRewardsLocked(
    combat *Combat,
    defeatedNPCs []NPC,
) error {
    if combat == nil {
        return ErrInvalidArgument
    }

    for _, npc := range defeatedNPCs {
        for _, itemID := range npc.Rewards {
            if _, exists := w.items[itemID]; !exists {
                return fmt.Errorf("%w: %q", ErrItemNotFound, itemID)
            }
        }
    }

    return nil
}