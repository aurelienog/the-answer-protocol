package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) View() string {
    if m.appState == StateAuthentication {
        return m.renderAuthentication()
    }

    return m.renderView()
}

func (m Model) renderAuthentication() string {
    title := layout.TitleStyle.Render("THE ANSWER PROTOCOL")

    content := lipgloss.JoinVertical(
        lipgloss.Left,
        title,
        "",
        "Welcome, adventurer!",
        "",
        "Enter your username:",
        "",
        layout.InputStyle.Render("> "+m.input),
        "",
        layout.StatusStyle.Render(m.status),
    )

    return layout.PanelStyle.Render(content)
}

func (m Model) renderView() string {
	if m.terminalTooSmall {
		return layout.WarningStyle.Render(
			"Terminal too small. Please enlarge your terminal.",
		)
	}

	header := layout.RenderHeader(
		m.username,
		m.playerStatus.HP,
		m.playerStatus.MaxHP,
	)

	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth

	eventLog := m.renderEventLog(leftWidth)
	chat := m.renderChat(rightWidth)

	bottomPanels := lipgloss.JoinHorizontal(
		lipgloss.Top,
		eventLog,
		chat,
	)
	input := layout.InputStyle.Render("> " + m.input)
	status := layout.StatusStyle.Render(m.status)

	bodyHeight := m.height -
		lipgloss.Height(header) -
		lipgloss.Height(bottomPanels) -
		lipgloss.Height(input) -
		lipgloss.Height(status)

	if bodyHeight < 0 {
		bodyHeight = 0
	}

	sidebarContent := ""
	if m.showSidebar {
		sidebarContent = m.sidebarViewport.View()
	}

	body := layout.RenderMain(
		m.mainViewport.View(),
		sidebarContent,
		m.width,
		bodyHeight,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		body,
		bottomPanels,
		input,
		status,
	)
}

func (m Model) renderWorld() string {
	return renderRoom(
		m.currentRoom,
		m.mainViewport.Width,
		m.mainViewport.Height,
	)
}

func (m Model) renderInventory() string {
	return "Inventory View"
}

func (m Model) renderCharacter() string {
	return "Character View"
}

func (m Model) renderQuests() string {
	return "Quests View"
}

func (m Model) renderGroup() string {
	return "Group View"
}

func (m Model) renderCombat() string {
	return "Combat View"
}


func (m Model) renderEventLog(width int) string {
    title := layout.TitleStyle.Render("EVENT LOG")

    var content strings.Builder
    content.WriteString(title)
    content.WriteString("\n")

    visibleEvents := m.eventLogHeight -
        layout.PanelHeightOffset -
        1

    if visibleEvents < 0 {
        visibleEvents = 0
    }

    start := 0
    if len(m.eventLog) > visibleEvents {
        start = len(m.eventLog) - visibleEvents
    }

    for _, event := range m.eventLog[start:] {
        content.WriteString(event)
        content.WriteString("\n")
    }

    contentWidth := width - layout.PanelWidthOffset
    if contentWidth < 0 {
        contentWidth = 0
    }

    return layout.PanelStyle.
        Width(contentWidth).
        Height(m.eventLogHeight - layout.PanelHeightOffset).
        Render(content.String())
}
