package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"the-answer-protocol/frontend/cli/ui/layout"
)

const (
	worldMapNodeWidth = 23
	worldMapGap       = 4
)

type worldMapNode struct {
	ID   string
	Name string
	Row  int
	Col  int
}

var worldMapNodes = []worldMapNode{
	{ID: "forest_edge", Name: "Whispering Forest", Row: 0, Col: 1},
	{ID: "town_gate", Name: "North Town Gate", Row: 1, Col: 1},
	{ID: "blacksmith_forge", Name: "Blacksmith Forge", Row: 2, Col: 0},
	{ID: "market_street", Name: "Market Street", Row: 2, Col: 1},
	{ID: "apothecary", Name: "Apothecary", Row: 2, Col: 2},
	{ID: "harbor_square", Name: "Harbor Square", Row: 3, Col: 1},
	{ID: "tavern", Name: "The Salty Siren", Row: 3, Col: 2},
	{ID: "docks", Name: "The Great Pier", Row: 4, Col: 1},
	{ID: "warehouse", Name: "Warehouse", Row: 4, Col: 2},
}

func (m Model) renderWorld() string {
	currentRoomID := m.world.Room.Room.ID

	title := layout.TitleStyle.Render("✧ ARVENDALE WORLD MAP ✧")

	var rows []string

	for row := 0; row <= 4; row++ {
		var nodes []string

		for col := 0; col <= 2; col++ {
			node := worldMapNodeAt(row, col)

			if node == nil {
				nodes = append(nodes, strings.Repeat(" ", worldMapNodeWidth))
				continue
			}

			nodes = append(
				nodes,
				renderWorldNode(*node, currentRoomID == node.ID),
			)
		}

		rows = append(
			rows,
			lipgloss.JoinHorizontal(lipgloss.Top, nodes...),
		)

		// if row < 4 {
		// 	rows = append(rows, renderWorldConnections(row))
		// }
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		strings.Join(rows, "\n"),
		"",
		layout.StatusStyle.Render(
			"Current location: "+m.world.Room.Room.Name,
		),
	)
}

func worldMapNodeAt(row, col int) *worldMapNode {
	for i := range worldMapNodes {
		if worldMapNodes[i].Row == row &&
			worldMapNodes[i].Col == col {
			return &worldMapNodes[i]
		}
	}

	return nil
}

func renderWorldNode(node worldMapNode, current bool) string {
	borderColor := layout.Muted

	if current {
		borderColor = layout.Primary
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	name := node.Name
	maxNameWidth := worldMapNodeWidth - 4

	if len(name) > maxNameWidth {
		name = name[:maxNameWidth]
	}

	padding := maxNameWidth - len(name)
	leftPadding := padding / 2
	rightPadding := padding - leftPadding

	top := borderStyle.Render(
		"+" + strings.Repeat("-", worldMapNodeWidth-2) + "+",
	)

	middle := borderStyle.Render("|") +
		strings.Repeat(" ", leftPadding+1) +
		name +
		strings.Repeat(" ", rightPadding+1) +
		borderStyle.Render("|")

	bottom := borderStyle.Render(
		"+" + strings.Repeat("-", worldMapNodeWidth-2) + "+",
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		top,
		middle,
		bottom,
	)
}

func renderWorldConnections(row int) string {
	vertical := lipgloss.NewStyle().
		Foreground(layout.Muted).
		Render("|")

	spaces := strings.Repeat(" ", worldMapNodeWidth)
	segment := strings.Repeat(" ", worldMapGap)

	var columns []string

	for col := 0; col <= 2; col++ {
		node := worldMapNodeAt(row, col)
		next := worldMapNodeAt(row+1, col)

		if node != nil && next != nil {
			columns = append(
				columns,
				lipgloss.PlaceHorizontal(
					worldMapNodeWidth,
					lipgloss.Center,
					vertical,
				),
			)
		} else {
			columns = append(columns, spaces)
		}
	}

	_ = fmt.Sprintf // Remove fmt import if unused.

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		columns[0],
		segment,
		columns[1],
		segment,
		columns[2],
	)
}
