package layout

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderSeparator(width int) string {
	if width <= 0 {
		return ""
	}

	ornament := " ✧ "
	remaining := width - lipgloss.Width(ornament)

	if remaining <= 0 {
		return MutedStyle.Render(strings.Repeat("─", width))
	}

	left := remaining / 2
	right := remaining - left

	return MutedStyle.Render(strings.Repeat("─", left)) +
		TitleStyle.Render(ornament) +
		MutedStyle.Render(strings.Repeat("─", right))
}

func RenderVerticalSeparator(height int) string {
	if height <= 0 {
		return ""
	}

	lines := make([]string, height)
	middle := height / 2

	for i := range lines {
		if i == middle {
			lines[i] = TitleStyle.Render("✧")
		} else {
			lines[i] = MutedStyle.Render("│")
		}
	}

	return strings.Join(lines, "\n")
}