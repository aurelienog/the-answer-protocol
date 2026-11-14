package layout

import "github.com/charmbracelet/lipgloss"

var (
	mainViewStyle = PanelStyle
	sidebarStyle  = PanelStyle
)

func RenderMain(
	mainContent string,
	sidebarContent string,
	mainWidth int,
	sidebarWidth int,
	height int,
	sidebarVisible bool,
) string {
	contentHeight := maxInt(0, height-PanelHeightOffset)

	main := mainViewStyle.
		Width(maxInt(0, mainWidth-PanelWidthOffset)).
		Height(contentHeight).
		Render(mainContent)

	if !sidebarVisible {
		return main
	}

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