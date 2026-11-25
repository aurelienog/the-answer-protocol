package ui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"the-answer-protocol/frontend/cli/ui"
)

func TestEventsCommandOpensScrollableHistory(t *testing.T) {
	const eventCount = 105

	model := ui.NewModel(newLogHistoryClient(t, eventCount))
	model, cmd := enterText(t, model, "Alice")
	if cmd == nil {
		t.Fatal("authentication command is nil")
	}
	model, cmd = updateModel(t, model, cmd())
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 120, Height: 40})

	for range eventCount {
		event := nextMessage(t, cmd)
		model, cmd = updateModel(t, model, event)
	}

	model, cmd = enterText(t, model, "EVENTS")
	if cmd == nil {
		t.Fatal("EVENTS did not start the event view transition")
	}
	model, cmd = updateModel(t, model, cmd())
	if cmd == nil {
		t.Fatal("event view transition did not schedule completion")
	}
	model, _ = updateModel(t, model, cmd())

	view := model.View()
	if !strings.Contains(view, "EVENT HISTORY") {
		t.Fatalf("EVENTS did not open the event history view: %q", view)
	}
	if !strings.Contains(view, "Notice update: event-005") {
		t.Errorf("event history = %q, want oldest retained event-005", view)
	}
	if strings.Contains(view, "Notice update: event-004") {
		t.Errorf("event history contains event-004, which should be outside the 100-event limit")
	}

	model, _ = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEnd})
	if view := model.View(); !strings.Contains(view, "Notice update: event-104") {
		t.Errorf("bottom of event history = %q, want newest event-104", view)
	}
}

func TestEventHistoryShowsConciseSummaries(t *testing.T) {
	events := []string{
		"EVT ITEM DROPPED item.rope",
		"EVT COMBAT DAMAGE attacker=Alice target=Dock Warden amount=12 remaining_hp=28",
		`EVT COMBAT STARTED combat=combat:2 room=loc.forest_edge participants=[{"id":"Bob","name":"Bob","is_player":true,"hp":100,"max_hp":100,"alive":true}]`,
		"EVT QUEST COMPLETED quest=quest.explore_arvendale",
	}
	model := ui.NewModel(newHistoryClient(t, len(events), func(index int) string {
		return events[index]
	}))
	model, cmd := enterText(t, model, "Alice")
	if cmd == nil {
		t.Fatal("authentication command is nil")
	}
	model, cmd = updateModel(t, model, cmd())
	model, _ = updateModel(t, model, tea.WindowSizeMsg{Width: 120, Height: 40})

	for range events {
		event := nextMessage(t, cmd)
		model, cmd = updateModel(t, model, event)
	}

	model, cmd = enterText(t, model, "EVENTS")
	if cmd == nil {
		t.Fatal("EVENTS did not start the event view transition")
	}
	model, cmd = updateModel(t, model, cmd())
	if cmd == nil {
		t.Fatal("event view transition did not schedule completion")
	}
	model, _ = updateModel(t, model, cmd())

	view := model.View()
	for _, expected := range []string{
		"Item dropped: rope",
		"Alice dealt 12 damage to Dock Warden",
		"Combat started in forest edge",
		"Quest completed: explore arvendale",
	} {
		if !strings.Contains(view, expected) {
			t.Errorf("event history = %q, want %q", view, expected)
		}
	}
	for _, payload := range []string{
		"EVT ITEM DROPPED",
		"remaining_hp=",
		"participants=",
		`"id":"Bob"`,
		"EVT QUEST COMPLETED",
	} {
		if strings.Contains(view, payload) {
			t.Errorf("event history contains raw payload %q: %q", payload, view)
		}
	}
}
