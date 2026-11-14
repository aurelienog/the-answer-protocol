package layout

import (
	"github.com/charmbracelet/lipgloss"
)

func MeasureContent(content string, width int) Size {
	if width <= 0 {
		return Size{}
	}

	return Size{
		Width:  width,
		Height: lipgloss.Height(lipgloss.NewStyle().Width(width).Render(content)),
	}
}

func MeasurePanelContent(content string, width int) Size {
	contentWidth := maxInt(0, width-PanelWidthOffset)
	size := MeasureContent(content, contentWidth)

	return Size{
		Width:  width,
		Height: size.Height + PanelHeightOffset,
	}
}

func MeasureHeader(playerName string, hp, maxHP, width int) Size {
	return MeasurePanelContent(
		HeaderContent(playerName, hp, maxHP),
		width,
	)
}

func MeasureStatus(content string, width int) Size {
	return MeasurePanelContent(content, width)
}

func MeasureAvailableCommands(content string, width int) Size {
	return MeasurePanelContent(content, width)
}