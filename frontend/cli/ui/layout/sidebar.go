package layout

import (
	"github.com/charmbracelet/lipgloss"
)

var sidebarStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	Padding(1, 2)

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