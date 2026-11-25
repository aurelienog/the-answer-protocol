package ui

import (
	"encoding/json"
	"strings"
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
	m.ui.PendingRefresh = RefreshNone
	return nil
}

func handleGroupInvitationsResponse(m *Model, payload string) error {
	var invitations []GroupInvitation

	if err := json.Unmarshal([]byte(payload), &invitations); err != nil {
		return err
	}

	m.player.Invitations = invitations
	m.ui.PendingRefresh = RefreshNone

	return nil
}

func removeInvitationByName(
	invitations []GroupInvitation,
	groupName string,
) []GroupInvitation {
	result := make([]GroupInvitation, 0, len(invitations))

	for _, invitation := range invitations {
		if !strings.EqualFold(invitation.Name, groupName) {
			result = append(result, invitation)
		}
	}

	return result
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

func handleUseResponse(m *Model, payload string) error {
	applyItemUseChanges(m, parsePayloadFields(payload))
	return nil
}

func handleQuestResponse(m *Model, payload string) error {
	var quest QuestSummary

	return json.Unmarshal([]byte(payload), &quest)
}

func upsertQuest(m *Model, quest Quest) {
	for i := range m.player.Quests {
		if m.player.Quests[i].ID == quest.ID {
			m.player.Quests[i] = quest
			return
		}
	}

	m.player.Quests = append(m.player.Quests, quest)
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
