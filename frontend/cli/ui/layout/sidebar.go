package layout

import "github.com/charmbracelet/lipgloss"

var sidebarStyle = PanelStyle

func RenderSidebar(questContent string, groupContent string) string {
    content := lipgloss.JoinVertical(
        lipgloss.Left,
        TitleStyle.Render(IconQuest + " QUESTS"),
        questContent,
        TitleStyle.Render(IconGroup + " GROUP"),
        groupContent,
    )

    return sidebarStyle.Render(content)
}