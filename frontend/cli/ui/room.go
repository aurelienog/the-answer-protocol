
package ui

import (
	"fmt"
	"sort"
	"strings"

	"the-answer-protocol/frontend/cli/ui/layout"
	"the-answer-protocol/internal/world"
)

func renderRoom(response LookResponse) string {
	var content strings.Builder

	content.WriteString(renderLocation(response))
	content.WriteString(renderPlayers(response.Players))
	content.WriteString(renderNPCs(response.NPCs))
	content.WriteString(renderObjects(response.Items))
	content.WriteString(renderExits(response.Room.Exits))

	return content.String()
}

func renderLocation(response LookResponse) string {
    var content strings.Builder

    content.WriteString(
        layout.TitleStyle.Render(
            layout.IconLocation + " LOCATION: " + response.Room.Name,
        ),
    )
    content.WriteString("\n\n")

    art := layout.GetRoomASCIIArt(response.Room.Name)
    if art != "" {
        content.WriteString(layout.RenderASCIIArt(art, 42))
        content.WriteString("\n\n")
    }

    content.WriteString(response.Room.Description)
    content.WriteString("\n\n")

    return content.String()
}

func renderPlayers(players []string) string {
	var content strings.Builder

	content.WriteString(layout.TitleStyle.Render(layout.IconPlayer + "  " + "PLAYERS"))
	content.WriteString("\n")

	if len(players) == 0 {
		content.WriteString(layout.StatusStyle.Render("  No players here"))
		content.WriteString("\n")
	} else {
		for _, player := range players {
			content.WriteString(fmt.Sprintf(
				"  %s %s\n",
				layout.SuccessStyle.Render(layout.PlayerMarker),
				player,
			))
		}
	}

	content.WriteString("\n")
	return content.String()
}

func renderNPCs(npcs []world.NPCSummary) string {
	var content strings.Builder

	content.WriteString(layout.TitleStyle.Render(layout.IconNPC + " NPCs"))
	content.WriteString("\n")

	if len(npcs) == 0 {
		content.WriteString(layout.StatusStyle.Render("  No NPCs here"))
		content.WriteString("\n")
	} else {
		for _, npc := range npcs {
			npcType := layout.SuccessStyle.Render("[Friendly]")

			if npc.Hostile {
				npcType = layout.DangerStyle.Render("[Hostile]")
			}

			content.WriteString(fmt.Sprintf(
				"  %s %s %s\n",
				layout.InfoStyle.Render(layout.NpcMarker),
				npc.Name,
				npcType,
			))
		}
	}

	content.WriteString("\n")
	return content.String()
}

func renderObjects(items []world.Item) string {
	var content strings.Builder

	content.WriteString(layout.TitleStyle.Render(layout.IconInventory + " OBJECTS"))
	content.WriteString("\n")

	if len(items) == 0 {
		content.WriteString(layout.StatusStyle.Render("  No objects here"))
		content.WriteString("\n")
	} else {
		for _, item := range items {
			content.WriteString(fmt.Sprintf(
				"  %s %s %s\n",
				layout.AccentStyle.Render(layout.ObjectMarker),
				item.Name,
				layout.StatusStyle.Render("["+string(item.Type)+"]"),
			))
		}
	}

	content.WriteString("\n")
	return content.String()
}

func renderExits(exits map[string]string) string {
	var content strings.Builder

	content.WriteString(layout.TitleStyle.Render(layout.IconExit + " EXITS"))
	content.WriteString("\n")

	directions := make([]string, 0, len(exits))
	for direction := range exits {
		directions = append(directions, direction)
	}
	sort.Strings(directions)

	if len(directions) == 0 {
		content.WriteString(layout.StatusStyle.Render("  No exits"))
		content.WriteString("\n")
	} else {
		for _, direction := range directions {
			content.WriteString(fmt.Sprintf(
				"  %s %s -> %s\n",
				layout.InfoStyle.Render(layout.ExitMarker),
				layout.TitleStyle.Render(strings.ToUpper(direction)),
				exits[direction],
			))
		}
	}

	return content.String()
}
