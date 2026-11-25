package ui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"the-answer-protocol/frontend/cli/ui"
)

func TestAuthenticationPanelUsesTerminalWidth(t *testing.T) {
	const terminalWidth = 100

	model := ui.NewModel(nil)
	model, _ = updateModel(t, model, tea.WindowSizeMsg{
		Width:  terminalWidth,
		Height: 30,
	})

	for _, line := range strings.Split(model.View(), "\n") {
		if got := lipgloss.Width(line); got != terminalWidth {
			t.Errorf("authentication panel line width = %d, want %d: %q", got, terminalWidth, line)
		}
	}
}

func TestAuthenticationShowsSmallTerminalWarning(t *testing.T) {
	model := ui.NewModel(nil)
	model, _ = updateModel(t, model, tea.WindowSizeMsg{
		Width:  40,
		Height: 30,
	})

	view := model.View()
	if !strings.Contains(view, "Terminal too small") {
		t.Fatalf("authentication view = %q, want small-terminal warning", view)
	}
	if strings.Contains(view, "THE ANSWER PROTOCOL") {
		t.Fatalf("authentication view = %q, want warning instead of panel", view)
	}
}
