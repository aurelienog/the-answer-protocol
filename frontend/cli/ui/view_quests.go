package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) renderQuests(detailed bool, availableWidth int) string {
	title := layout.TitleStyle.Render(layout.IconQuest + " QUESTS")

	if len(m.player.Quests) == 0 {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			layout.MutedStyle.Render("No quests discovered"),
		)
	}

	sections := []string{
		title,
		"",
	}

	for i, quest := range m.player.Quests {
		if i > 0 {
			sections = append(sections, "")
		}

		sections = append(
			sections,
			renderQuest(quest, detailed, availableWidth),
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		sections...,
	)
}

func renderQuest(quest Quest, detailed bool, availableWidth int) string {
	title := layout.TitleStyle.Render(quest.Name)
	status := renderQuestStatus(quest.Status)

	lines := []string{
		title,
		fmt.Sprintf("%s %s", layout.MutedStyle.Render("Status:"), status),
	}

	if detailed {
		if quest.GiverNPCName != "" {
			lines = append(
				lines,
				fmt.Sprintf("%s %s",
					layout.MutedStyle.Render("Giver:"),
					layout.InfoStyle.Render(quest.GiverNPCName),
				),
			)
		}
	}

	if quest.Description != "" {
		description := quest.Description

		if detailed {
			description = layout.WrapText(layout.MutedStyle.Render("Description: ")+description, availableWidth)
		} else {
			description = strings.SplitN(layout.MutedStyle.Render(description), "\n", 2)[0]
		}

		lines = append(lines, description)
	}

	if quest.Progress != "" {
		lines = append(
			lines,
			layout.InfoStyle.Render("Progress: "+quest.Progress),
		)
	}

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
		return layout.MutedStyle.Render(status)
	}
}
