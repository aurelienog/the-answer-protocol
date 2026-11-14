package layout

import "github.com/charmbracelet/lipgloss"

func RenderSidebar(questContent string, groupContent string) string {
    return lipgloss.JoinVertical(
        lipgloss.Left,
        TitleStyle.Render(IconQuest+" QUESTS"),
        questContent,
        TitleStyle.Render(IconGroup+" GROUP"),
        groupContent,
    )
}