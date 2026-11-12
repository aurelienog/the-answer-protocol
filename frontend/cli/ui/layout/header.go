package layout

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var headerStyle = lipgloss.NewStyle().
	Bold(true).
	Padding(0, 1).
	Foreground(lipgloss.Color("205"))

func RenderHeader(playerName string, hp int, maxHP int) string {
	title := headerStyle.Render("THE ANSWER PROTOCOL")

	player := fmt.Sprintf(
		"Player: %s | HP: %d/%d",
		playerName,
		hp,
		maxHP,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		player,
	)
}