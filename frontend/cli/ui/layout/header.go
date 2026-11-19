package layout

import (
	"github.com/charmbracelet/lipgloss"
)

func RenderHeader(
	username string,
	hp, maxHP, width int,
    ) string {
	content := HeaderContent(username, hp, maxHP)

	panelWidth := maxInt(0, width-PanelBorderWidth)

	return PanelStyle.
		Width(panelWidth).
		Render(content)
}

func HeaderContent(playerName string, hp, maxHP int) string {
    title := TitleStyle.Render("THE ANSWER PROTOCOL")
    player := "Player: " + playerName
    health := "HP  " + RenderHealthBar(hp, maxHP, 20)

    return lipgloss.JoinVertical(
        lipgloss.Left,
        title,
        player,
        health,
    )
}