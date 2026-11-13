
package layout

import (
	"fmt"
	"strings"
)

func RenderProgressBar(current, maximum, width int) string {
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

	filled := current * width / maximum
	empty := width - filled

	bar := SuccessStyle.Render(strings.Repeat("█", filled)) +
		MutedStyle.Render(strings.Repeat("░", empty))

	percentage := current * 100 / maximum

	return fmt.Sprintf(
		"%s %d/%d (%d%%)",
		bar,
		current,
		maximum,
		percentage,
	)
}
