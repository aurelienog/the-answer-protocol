package layout

import (
	"github.com/charmbracelet/lipgloss"
)

func RenderHeader(playerName string, hp, maxHP, width int) string {
    return PanelStyle.
        Width(maxInt(0, width-PanelWidthOffset)).
        Render(HeaderContent(playerName, hp, maxHP))
}

func HeaderContent(playerName string, hp, maxHP int) string {
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