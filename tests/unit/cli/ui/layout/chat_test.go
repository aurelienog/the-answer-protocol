package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderChatKeepsAllocatedHeightAcrossFilters(t *testing.T) {
	model := NewModel(nil)
	model.communication.ChatLog = []ChatMessage{
		{
			Scope:      "GLOBAL",
			PlayerName: "player",
			Message:    strings.Repeat("long message ", 30),
		},
		{
			Scope:      "ROOM",
			PlayerName: "player",
			Message:    "hello",
		},
	}

	for _, filter := range []ChatFilter{
		ChatFilterAll,
		ChatFilterGlobal,
		ChatFilterRoom,
	} {
		model.communication.ChatFilter = filter
		panel := model.renderChat(40, 8)

		if got := lipgloss.Height(panel); got != 8 {
			t.Errorf("filter %d: panel height = %d, want 8", filter, got)
		}

		if got := strings.Count(panel, "CHAT"); got != 1 {
			t.Errorf("filter %d: CHAT title count = %d, want 1", filter, got)
		}
	}
}

func TestRenderChatPrioritizesRecentMessages(t *testing.T) {
	model := NewModel(nil)
	model.communication.ChatLog = []ChatMessage{
		{Scope: "ROOM", PlayerName: "player", Message: "older chat message"},
		{
			Scope:      "ROOM",
			PlayerName: "player",
			Message:    strings.Repeat("intermediate message ", 30),
		},
		{Scope: "ROOM", PlayerName: "player", Message: "latest chat message"},
	}

	panel := model.renderChat(40, 8)

	if !strings.Contains(panel, "latest chat message") {
		t.Error("chat panel does not show the latest message")
	}

	if strings.Contains(panel, "older chat message") {
		t.Error("chat panel shows an older message instead of prioritizing recent messages")
	}
}

func TestRenderEventLogPrioritizesRecentEvents(t *testing.T) {
	model := NewModel(nil)
	model.communication.EventLog = []string{
		"older event",
		strings.Repeat("intermediate event ", 30),
		"latest event",
	}

	panel := model.renderEventLog(40, 8)

	if !strings.Contains(panel, "latest event") {
		t.Error("event log does not show the latest event")
	}

	if strings.Contains(panel, "older event") {
		t.Error("event log shows an older event instead of prioritizing recent events")
	}
}
