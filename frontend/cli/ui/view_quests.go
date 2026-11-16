
package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) renderQuests() string {
	if len(m.world.Quests) == 0 {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			layout.TitleStyle.Render(layout.IconQuest+" QUESTS"),
			"",
			layout.StatusStyle.Render("No quests discovered"),
		)
	}

	sections := []string{
		layout.TitleStyle.Render(layout.IconQuest+" QUESTS"),
	}

	for _, quest := range m.world.Quests {
		sections = append(sections, renderQuest(quest))
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		sections...,
	)
}

func renderQuest(quest Quest) string {
    title := layout.TitleStyle.Render(quest.Name)
    status := renderQuestStatus(quest.Status)

    lines := []string{
        title,
        fmt.Sprintf("Status: %s", status),
    }

    if quest.Description != "" {
        lines = append(lines, layout.StatusStyle.Render(quest.Description))
    }

    if quest.Progress != "" {
        lines = append(lines, layout.InfoStyle.Render("Progress: "+quest.Progress))
    }

    lines = append(lines, "")

    return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func renderQuestStatus(status string) string {
	switch status {
	case "available":
		return layout.InfoStyle.Render(status)
	case "active":
		return layout.WarningStyle.Render(status)
	case "completed":
		return layout.SuccessStyle.Render(status)
	case "locked":
		return layout.MutedStyle.Render(status)
	default:
		return layout.StatusStyle.Render(status)
	}
}
