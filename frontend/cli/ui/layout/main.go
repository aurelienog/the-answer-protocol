package layout

import "github.com/charmbracelet/lipgloss"

const (
    minWidthForSidebar = 90

    panelBorderWidth  = 2
    panelBorderHeight = 2
    panelPaddingWidth  = 4
    panelPaddingHeight = 2

    panelWidthOffset  = panelBorderWidth + panelPaddingWidth
    panelHeightOffset = panelBorderHeight + panelPaddingHeight
)

var mainViewStyle = PanelStyle

func RenderMain(mainContent, sidebarContent string, width, height int) string {
    if width < minWidthForSidebar {
        return mainViewStyle.
            Width(width - panelWidthOffset).
            Height(height - panelHeightOffset).
            Render(mainContent)
    }

    mainWidth := width * 75 / 100
    sidebarWidth := width - mainWidth

    main := mainViewStyle.
        Width(mainWidth - panelWidthOffset).
        Height(height - panelHeightOffset).
        Render(mainContent)

    sidebar := sidebarStyle.
        Width(sidebarWidth - panelWidthOffset).
        Height(height - panelHeightOffset).
        Render(sidebarContent)

    return lipgloss.JoinHorizontal(
        lipgloss.Top,
        main,
        sidebar,
    )
}