package ui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func handleLookResponse(m *Model, payload string) error {
	var response LookResponse

	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		return err
	}

	m.world.Room = response

	m.ui.PendingRefresh = RefreshNone

	return nil
}

func handleMoveResponse(m *Model, payload string) error {
	return nil
}

func handleTalkResponse(m *Model, payload string) error {
	var conversation NPCConversation

	if err := json.Unmarshal([]byte(payload), &conversation); err != nil {
		return err
	}

	m.world.Conversation = &conversation

	return nil
}

func handleDropResponse(_ *Model, payload string) error {
	const prefix = "dropped="

	if !strings.HasPrefix(payload, prefix) {
		return fmt.Errorf("unexpected DROP response: %s", payload)
	}

	itemID := strings.TrimSpace(strings.TrimPrefix(payload, prefix))
	if itemID == "" {
		return fmt.Errorf("DROP response contains an empty item ID")
	}

	return nil
}

func handleTakeResponse(m *Model, payload string) error {
	const prefix = "taken="

	if !strings.HasPrefix(payload, prefix) {
		return fmt.Errorf("unexpected TAKE response: %s", payload)
	}

	itemID := strings.TrimSpace(strings.TrimPrefix(payload, prefix))
	if itemID == "" {
		return fmt.Errorf("TAKE response contains an empty item ID")
	}

	summary, style := formatItemEvent("TAKEN item=" + itemID)
	m.appendEventLog(style.Render(summary))

	return nil
}

func handleWhoResponse(m *Model, payload string) error {
	const prefix = "players="

	if !strings.HasPrefix(payload, prefix) {
		return fmt.Errorf("invalid WHO response")
	}

	count, err := strconv.Atoi(strings.TrimPrefix(payload, prefix))
	if err != nil {
		return err
	}

	if count < 0 {
		return fmt.Errorf("invalid player count")
	}

	m.world.OnlinePlayers = count
	return nil
}
