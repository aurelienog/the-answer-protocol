
package layout

import (
	"fmt"
	"strings"
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