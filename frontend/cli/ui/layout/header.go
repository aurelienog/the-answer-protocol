package layout

import (
    "github.com/charmbracelet/lipgloss"
)

func RenderHeader(
    username string,
    hp, maxHP, width int,
) string {
    return RenderHeaderWithHealth(
        username,
        float64(hp),
        maxHP,
        width,
    )
}

func RenderHeaderWithHealth(
    username string,
    hp float64,
    maxHP, width int,
) string {
    content := HeaderContentWithHealth(username, hp, maxHP)

    panelWidth := maxInt(0, width-PanelBorderWidth)

    return PanelStyle.
        Width(panelWidth).
        Render(content)
}

func HeaderContent(playerName string, hp, maxHP int) string {
    return HeaderContentWithHealth(
        playerName,
        float64(hp),
        maxHP,
    )
}

func HeaderContentWithHealth(
    playerName string,
    hp float64,
    maxHP int,
) string {
    title := TitleStyle.Render("THE ANSWER PROTOCOL")
    player := "Player: " + playerName
    health := "HP  " + RenderAnimatedHealthBar(hp, maxHP, 20)

    return lipgloss.JoinVertical(
        lipgloss.Left,
        title,
        player,
        health,
    )
}