package layout

import (
	"github.com/charmbracelet/lipgloss"
)

func RenderHeader(playerName string, hp int, maxHP int, width int) string {
	title := TitleStyle.Render("THE ANSWER PROTOCOL")
	player := "Player: " + playerName
	health := "HP  " + RenderProgressBar(hp, maxHP, 20)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		player,
		health,
	)

	return PanelStyle.
		Width(maxInt(0, width-PanelWidthOffset)).
		Render(content)
}