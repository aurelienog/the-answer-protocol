package ui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"the-answer-protocol/frontend/cli/ui"
)

func TestChatModelInitialView(t *testing.T) {
	model := ui.NewModel(nil)

	view := model.View()
	if !strings.Contains(view, "Welcome, adventurer!") {
		t.Fatalf("initial view = %q, want welcome screen", view)
	}
}

func TestChatModelAcceptsWindowResize(t *testing.T) {
	model := ui.NewModel(nil)

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if updated == nil {
		t.Fatal("Update(WindowSizeMsg) returned a nil model")
	}
	if view := updated.View(); view == "" {
		t.Fatal("resized view is empty")
	}
	if cmd != nil {
		t.Log("resize scheduled a follow-up command")
	}
}
