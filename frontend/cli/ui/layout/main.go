package layout

import "github.com/charmbracelet/lipgloss"

const minWidthForSidebar = 90

var mainViewStyle = lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    Padding(1, 2)

func RenderMain(mainContent, sidebarContent string, width int) string {
    if width < minWidthForSidebar {
        return mainViewStyle.
            Width(width - 6).
            Render(mainContent)
    }

    mainWidth := width * 75 / 100
    sidebarWidth := width - mainWidth

    main := mainViewStyle.
        Width(mainWidth - 6).
        Render(mainContent)

    sidebar := sidebarStyle.
        Width(sidebarWidth - 6).
        Render(sidebarContent)

    return lipgloss.JoinHorizontal(
        lipgloss.Top,
        main,
        sidebar,
    )
}