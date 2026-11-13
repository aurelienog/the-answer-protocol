package layout

import (
    "github.com/charmbracelet/lipgloss"
)

func RenderHeader(playerName string, hp int, maxHP int) string {
	title := TitleStyle.Render("THE ANSWER PROTOCOL")
	player := "Player: " + playerName

	health := "HP  " + RenderProgressBar(hp, maxHP, 20)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		player,
		health,
	)
}
