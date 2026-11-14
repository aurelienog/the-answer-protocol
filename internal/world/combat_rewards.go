package world

import "fmt"

// distributeCombatRewardsLocked distributes the rewards of all defeated NPCs
// to participating players who did not escape.
// The caller must hold w.mu.Lock().
func (w *World) distributeCombatRewardsLocked(
    combat *Combat,
    defeatedNPCs []NPC,
) ([]CombatReward, error) {
    if combat == nil {
        return nil, fmt.Errorf("%w: nil combat", ErrInvalidArgument)
    }

    // Validate every reward item before creating instances.
    for _, npc := range defeatedNPCs {
        for _, itemID := range npc.Rewards {
            if _, exists := w.items[itemID]; !exists {
                return nil, fmt.Errorf("%w: %q", ErrItemNotFound, itemID)
            }
        }
    }

    // Prepare rewards without modifying player inventories.
    pendingRewards := make(map[string][]string)
    rewardsByPlayer := make(map[string]int)
    rewards := make([]CombatReward, 0)

    previousCounter := w.nextItemInstanceID
    createdInstances := make([]string, 0)

    for _, participant := range combat.Participants {
        if !participant.IsPlayer ||
            participant.Escaped ||
            !participant.Connected {
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
                    return nil, err
                }

                createdInstances = append(createdInstances, instanceID)
                pendingRewards[participant.ID] = append(
                    pendingRewards[participant.ID],
                    instanceID,
                )

                // Record the item definition for the notification.
                rewardIndex, exists := rewardsByPlayer[participant.ID]
                if !exists {
                    rewardIndex = len(rewards)
                    rewardsByPlayer[participant.ID] = rewardIndex
                    rewards = append(rewards, CombatReward{
                        PlayerID: participant.ID,
                        Items:    make([]Item, 0),
                    })
                }

                rewards[rewardIndex].Items = append(
                    rewards[rewardIndex].Items,
                    w.items[itemID],
                )
            }
        }
    }

    // Commit all rewards only after successful preparation.
    if len(pendingRewards) > 0 && w.playerInventories == nil {
        w.playerInventories = make(map[string][]string)
    }

    for playerID, instanceIDs := range pendingRewards {
        w.playerInventories[playerID] = append(
            w.playerInventories[playerID],
            instanceIDs...,
        )
    }

    return rewards, nil
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