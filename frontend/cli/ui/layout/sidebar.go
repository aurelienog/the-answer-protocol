package layout

import (
    "strings"
    "github.com/charmbracelet/lipgloss"
)
func RenderSidebar(questContent string, groupContent string, width int) string {
    separatorWidth := (width * 60) / 100
    padding := strings.Repeat("\n", 1)

    separator := lipgloss.PlaceHorizontal(
        width,
        lipgloss.Center,
        RenderSeparator(separatorWidth),
    )

    return lipgloss.JoinVertical(
        lipgloss.Left,
        questContent,
        padding,
        separator,
        padding,
        groupContent,
        padding,
    )
}