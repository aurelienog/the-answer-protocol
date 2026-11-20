package ui
import (
	"encoding/json"
)
func handleStatusResponse(m *Model, payload string) error {
	var status PlayerStatus

	if err := json.Unmarshal([]byte(payload), &status); err != nil {
		return err
	}

	m.player.Status = status
	m.ui.PendingRefresh = RefreshNone

	return nil
}

func handleGroupInfoResponse(m *Model, payload string) error {
	var group Group

	if err := json.Unmarshal([]byte(payload), &group); err != nil {
		return err
	}

	m.world.Group = group

	return nil
}

func handleInventoryResponse(m *Model, payload string) error {
	var inventory []InventoryItem

	if err := json.Unmarshal([]byte(payload), &inventory); err != nil {
		return err
	}

	m.player.Inventory = inventory
	m.ui.PendingRefresh = RefreshNone

	return nil
}

func handleQuestResponse(m *Model, payload string) error {
    var quest QuestSummary

    return json.Unmarshal([]byte(payload), &quest)
}

func handleQuestsResponse(m *Model, payload string) error {
    var quests []Quest

    if err := json.Unmarshal([]byte(payload), &quests); err != nil {
        return err
    }

    m.player.Quests = quests

	m.ui.PendingRefresh = RefreshNone

    return nil
}
