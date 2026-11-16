package ui

import "encoding/json"

func handleStatusResponse(m *Model, payload string) error {
	var status PlayerStatus

	if err := json.Unmarshal([]byte(payload), &status); err != nil {
		return err
	}

	m.player.Status = status
	m.setView(ViewCharacter)

	return nil
}

func handleGroupInfoResponse(m *Model, payload string) error {
	var group Group

	if err := json.Unmarshal([]byte(payload), &group); err != nil {
		return err
	}

	m.world.Group = group
	m.setView(ViewGroup)

	return nil
}

func handleInventoryResponse(m *Model, payload string) error {
	var inventory []InventoryItem

	if err := json.Unmarshal([]byte(payload), &inventory); err != nil {
		return err
	}

	m.player.Inventory = inventory

	if m.ui.PendingRefresh != RefreshInventory {
		m.setView(ViewInventory)
	}

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

	if m.ui.PendingRefresh != RefreshQuests {
		m.setView(ViewQuests)
	}

	m.ui.PendingRefresh = RefreshNone

    return nil
}
