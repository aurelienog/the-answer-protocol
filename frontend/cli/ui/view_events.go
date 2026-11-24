package ui

import (
	"github.com/charmbracelet/lipgloss"

	"the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) renderEventsView(width int) string {
	if width <= 0 {
		return ""
	}

	sections := []string{
		layout.TitleStyle.Render("EVENT HISTORY"),
		"",
	}

	if len(m.communication.EventLog) == 0 {
		sections = append(
			sections,
			layout.MutedStyle.Render("No events recorded yet"),
		)
	} else {
		for _, entry := range m.communication.EventLog {
			sections = append(
				sections,
				layout.WrapText(entry, width),
				"",
			)
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		sections...,
	)
}
