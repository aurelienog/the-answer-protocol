package layout

import "github.com/charmbracelet/lipgloss"

const (
    EventLogMinEntries = 3
    EventLogMaxHeight  = 12
)

var mainViewStyle = PanelStyle

func RenderMain(mainContent, sidebarContent string, width, height int) string {
	contentHeight := maxInt(
		0,
		height-PanelHeightOffset,
	)

	// Si la sidebar no está disponible, Main View ocupa todo el ancho.
	if width < MinWidthForSidebar || sidebarContent == "" {
		return mainViewStyle.
			Width(maxInt(0, width-PanelWidthOffset)).
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

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		main,
		sidebar,
	)
}

func maxInt(a, b int) int {
    if a > b {
        return a
    }
    return b
}