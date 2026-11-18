package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) renderTalk() string {
	conversation := m.world.Conversation
	width := m.ui.MainViewport.Width

	if conversation == nil {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			layout.TitleStyle.Render("CONVERSATION"),
			"",
			layout.StatusStyle.Render("No active conversation"),
		)
	}

	sections := []string{
		layout.TitleStyle.Render("CONVERSATION"),
		"",
		layout.TitleStyle.Render(conversation.NPC.Name),
	}

	sections = append(sections, "")

	// Display the NPC dialogue.
	for _, line := range conversation.Dialogue {
		sections = append(
			sections,
			layout.WrapText(line, width),
		)
	}

	// Display quest information when the NPC offers one.
	if conversation.Quest != nil {
		sections = append(
			sections,
			"",
			layout.RenderSeparator(width),
			layout.TitleStyle.Render("QUEST\n"),
			layout.TitleStyle.Render(conversation.Quest.Name),
		)

		if conversation.Quest.Description != "" {
			sections = append(
				sections,
				layout.WrapText(conversation.Quest.Description, width),
			)
		}

		if conversation.Quest.Progress != "" {
			sections = append(
				sections,
				layout.WrapText(
					"Progress: "+conversation.Quest.Progress,
					width,
				),
			)
		}

		sections = append(
			sections,
			fmt.Sprintf(
				"Status: %s",
				renderQuestStatus(conversation.QuestStatus),
			),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
