
package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) renderCharacter() string {
	sectionTitle := layout.TitleStyle.Render("CHARACTER")

	name := layout.TitleStyle.Render(m.player.Username)

	healthLabel := layout.StatusStyle.Render("Health")
	healthValue := fmt.Sprintf("%d / %d", m.player.Status.HP, m.player.Status.MaxHP)

	healthStyle := layout.SuccessStyle
	if m.player.Status.MaxHP > 0 &&
		m.player.Status.HP <= m.player.Status.MaxHP/4 {
		healthStyle = layout.DangerStyle
	}

	statusLabel := layout.StatusStyle.Render("Condition")
	statusValue := layout.StatusStyle.Render(m.player.Status.Status)

	healthLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		healthLabel,
		": ",
		healthStyle.Render(healthValue),
	)

	statusLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		statusLabel,
		": ",
		statusValue,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		sectionTitle,
		"",
		name,
		healthLine,
		statusLine,
	)
}
