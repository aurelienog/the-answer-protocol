package ui

import (
	"fmt"
	"sort"
	"strings"

	"the-answer-protocol/frontend/cli/ui/layout"
)

func renderRoom(response LookResponse, availableWidth int) string {
	content := renderLocation(response, availableWidth)
	content += renderPlayers(response.Players)
	content += renderNPCs(response.NPCs)
	content += renderObjects(response.Items)
	content += renderExits(response.Room.Exits)
	return content
}

func renderLocation(response LookResponse, availableWidth int) string {
	var content strings.Builder

	content.WriteString(layout.TitleStyle.Render(
		layout.IconLocation + " LOCATION: " + response.Room.Name,
	))
	content.WriteString("\n\n")
	content.WriteString(layout.WrapText(
		response.Room.Description,
		availableWidth,
	))
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

func renderNPCs(npcs []NPCSummary) string {
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

func renderObjects(items []Item) string {
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
				friendlyID(exits[direction]),
			))
		}
	}

	return content.String()
}
