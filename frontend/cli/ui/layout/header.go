package layout

import (
    "fmt"

    "github.com/charmbracelet/lipgloss"
)

func RenderHeader(playerName string, hp int, maxHP int) string {
    title := TitleStyle.Render("THE ANSWER PROTOCOL")

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