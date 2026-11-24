package ui
import (
	"encoding/json"
	"fmt"
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

func handleGroupJoinResponse(m *Model, payload string) error {
	fields := parsePayloadFields(payload)
	groupID := fields["group"]

	if groupID == "" {
		return fmt.Errorf("missing group ID in GROUP JOIN response")
	}

	m.player.Invitations = removeInvitationByID(
		m.player.Invitations,
		groupID,
	)

	m.ui.PendingRefresh = RefreshGroup

	return nil
}

func removeInvitationByID(
	invitations []GroupInvitation,
	groupID string,
) []GroupInvitation {
	result := make([]GroupInvitation, 0, len(invitations))

	for _, invitation := range invitations {
		if invitation.ID != groupID {
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
