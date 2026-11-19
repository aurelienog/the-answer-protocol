
package layout

import (
	"fmt"
	"strings"
	"math"
)

func RenderHealthBar(current, maximum, width int) string {
	if maximum <= 0 {
		return MutedStyle.Render("[No data]")
	}

	if current < 0 {
		current = 0
	}
	if current > maximum {
		current = maximum
	}
	if width < 1 {
		width = 1
	}

	percentage := current * 100 / maximum
	filled := current * width / maximum
	empty := width - filled

	healthStyle := SuccessStyle

	switch {
	case percentage <= 25:
		healthStyle = DangerStyle
	case percentage <= 50:
		healthStyle = WarningStyle
	}

	bar := healthStyle.Render(strings.Repeat("█", filled)) +
		MutedStyle.Render(strings.Repeat("░", empty))

	return fmt.Sprintf(
		"%s %d/%d (%d%%)",
		bar,
		current,
		maximum,
		percentage,
	)
}

func RenderAnimatedHealthBar(current float64, maximum, width int) string {
	if maximum <= 0 {
		return MutedStyle.Render("[No data]")
	}

	if current < 0 {
		current = 0
	}
	if current > float64(maximum) {
		current = float64(maximum)
	}
	if width < 1 {
		width = 1
	}

	percentage := int(current / float64(maximum) * 100)
	filled := int(math.Round(current / float64(maximum) * float64(width)))

	if filled > width {
		filled = width
	}

	empty := width - filled

	healthStyle := SuccessStyle

	switch {
	case percentage <= 25:
		healthStyle = DangerStyle
	case percentage <= 50:
		healthStyle = WarningStyle
	}

	bar := healthStyle.Render(strings.Repeat("█", filled)) +
		MutedStyle.Render(strings.Repeat("░", empty))

	return fmt.Sprintf(
		"%s %d/%d (%d%%)",
		bar,
		int(math.Round(current)),
		maximum,
		percentage,
	)
}