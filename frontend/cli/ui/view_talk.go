
package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) renderTalk() string {
	conversation := m.world.Conversation

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

	if conversation.NPC.Role != "" {
		sections = append(
			sections,
			layout.StatusStyle.Render(conversation.NPC.Role),
		)
	}

	sections = append(sections, "")

	// Display the NPC dialogue.
	for _, line := range conversation.Dialogue {
		sections = append(sections, line)
	}

	// Display quest information when the NPC offers one.
	if conversation.Quest != nil {
		sections = append(sections,
			"",
			layout.RenderSeparator(m.ui.MainViewport.Width),
			layout.TitleStyle.Render("QUEST"),
			layout.TitleStyle.Render(conversation.Quest.Name),
		)

		if conversation.Quest.Description != "" {
			sections = append(
				sections,
				layout.StatusStyle.Render(conversation.Quest.Description),
			)
		}

		if conversation.Quest.Progress != "" {
			sections = append(
				sections,
				layout.InfoStyle.Render(
					"Progress: "+conversation.Quest.Progress,
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

		if conversation.QuestStatus == "available" {
			sections = append(
				sections,
				layout.InfoStyle.Render(
					"Use QUEST <npc> to accept this quest",
				),
			)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
