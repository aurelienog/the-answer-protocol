package layout

import (
	"github.com/charmbracelet/lipgloss"
)

var sidebarStyle = PanelStyle

func RenderSidebar(groupContent string, questContent string) string {
    content := lipgloss.JoinVertical(
        lipgloss.Left,
        "GROUP",
        groupContent,
        "",
        "QUESTS",
        questContent,
    )

    return sidebarStyle.Render(content)
}