package layout

import "github.com/charmbracelet/lipgloss"

const (
    minWidthForSidebar = 90
)

var mainViewStyle = PanelStyle

func RenderMain(mainContent, sidebarContent string, width, height int) string {
    contentWidth := maxInt(0, width-PanelWidthOffset)
    contentHeight := maxInt(0, height-PanelHeightOffset)

    if width < minWidthForSidebar {
        return mainViewStyle.
            Width(contentWidth).
            Height(contentHeight).
            Render(mainContent)
    }

    mainWidth := width * 75 / 100
    sidebarWidth := width - mainWidth

    main := mainViewStyle.
        Width(maxInt(0, mainWidth-PanelWidthOffset)).
        Height(contentHeight).
        Render(mainContent)

    sidebar := sidebarStyle.
        Width(maxInt(0, sidebarWidth-PanelWidthOffset)).
        Height(contentHeight).
        Render(sidebarContent)

    return lipgloss.JoinHorizontal(lipgloss.Top, main, sidebar)
}

func maxInt(a, b int) int {
    if a > b {
        return a
    }
    return b
}