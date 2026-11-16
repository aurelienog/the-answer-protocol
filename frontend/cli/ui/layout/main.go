
package layout

import "github.com/charmbracelet/lipgloss"

func RenderMain(
	mainContent string,
	sidebarContent string,
	mainWidth int,
	sidebarWidth int,
	height int,
	sidebarVisible bool,
	panelFocused bool,
) string {
	contentHeight := maxInt(0, height-PanelHeightOffset)

	mainStyle := UnfocusedPanelStyle
	sidebarPanelStyle := UnfocusedPanelStyle

	if !panelFocused {
		mainStyle = FocusedPanelStyle
	} else {
		sidebarPanelStyle = FocusedPanelStyle
	}

	main := mainStyle.
		Width(maxInt(0, mainWidth-PanelBorderWidth)).
		Height(contentHeight).
		Render(mainContent)

	if !sidebarVisible {
		return main
	}

	sidebar := sidebarPanelStyle.
		Width(maxInt(0, sidebarWidth-PanelBorderWidth)).
		Height(contentHeight).
		Render(sidebarContent)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		main,
		sidebar,
	)
}
