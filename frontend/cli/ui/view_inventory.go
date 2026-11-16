
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"the-answer-protocol/frontend/cli/ui/layout"
)

const inventoryColumnGap = 1

func (m Model) renderInventory() string {
	width := m.ui.MainViewport.Width

	if width <= 0 {
		return ""
	}

	commonItems := make([]InventoryItem, 0)
	questItems := make([]InventoryItem, 0)
	equipmentItems := make([]InventoryItem, 0)

	for _, item := range m.player.Inventory {
		switch item.Type {
		case "common", "consumable":
			commonItems = append(commonItems, item)
		case "quest":
			questItems = append(questItems, item)
		case "equipment", "unique":
			equipmentItems = append(equipmentItems, item)
		}
	}

	columns := []inventoryColumn{
		{title: "COMMON", items: commonItems},
		{title: "QUEST", items: questItems},
		{title: "EQUIPMENT", items: equipmentItems},
	}

	columnCount := resolveInventoryColumnCount(width)

	columnWidth := (width - (columnCount-1)*inventoryColumnGap) / columnCount

	renderedColumns := make([]string, 0, len(columns))

	for _, column := range columns {
		renderedColumns = append(
			renderedColumns,
			renderInventoryColumn(column.title, column.items, columnWidth),
		)
	}

	var rows []string

	for i := 0; i < len(renderedColumns); i += columnCount {
		end := i + columnCount
		if end > len(renderedColumns) {
			end = len(renderedColumns)
		}

		row := lipgloss.JoinHorizontal(
			lipgloss.Top,
			renderInventoryColumns(renderedColumns[i:end])...,
		)

		rows = append(rows, row)
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

type inventoryColumn struct {
	title string
	items []InventoryItem
}

func renderInventoryColumns(columns []string) []string {
	result := make([]string, 0, len(columns)*2-1)

	for i, column := range columns {
		if i > 0 {
			result = append(result, strings.Repeat(" ", inventoryColumnGap))
		}

		result = append(result, column)
	}

	return result
}

func renderInventoryColumn(title string, items []InventoryItem, width int) string {
	titleStyle := layout.TitleStyle.Copy().Width(width)

	lines := []string{
		titleStyle.Render(title),
		layout.RenderSeparator(width),
	}

	if len(items) == 0 {
		lines = append(lines, layout.StatusStyle.Render("No items"))
	} else {
		for _, item := range items {
			name := item.Name
			if name == "" {
				name = item.ID
			}

			entry := fmt.Sprintf("%s x%d", name, item.Quantity)
			lines = append(lines, lipgloss.NewStyle().Width(width).Render(entry))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func resolveInventoryColumnCount(width int) int {
	switch {
	case width >= 3*layout.MinInventoryColumnWidth+
		2*inventoryColumnGap:
		return 3
	case width >= 2*layout.MinInventoryColumnWidth+
		inventoryColumnGap:
		return 2
	default:
		return 1
	}
}
