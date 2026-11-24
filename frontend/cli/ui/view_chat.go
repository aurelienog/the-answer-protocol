package ui

import (
	"github.com/charmbracelet/lipgloss"

	"the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) renderChatView(width int) string {
	if width <= 0 {
		return ""
	}

	sections := []string{
		layout.TitleStyle.Render("CHAT HISTORY"),
		m.renderChatFilter(),
		"",
	}

	messages := m.filteredChatMessages()
	if len(messages) == 0 {
		sections = append(
			sections,
			layout.MutedStyle.Render("No messages for this filter"),
		)
	} else {
		for _, message := range messages {
			sections = append(
				sections,
				layout.WrapText(renderChatMessage(message), width),
				"",
			)
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		sections...,
	)
}
