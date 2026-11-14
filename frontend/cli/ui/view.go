
package ui

import (
    "strings"

    "github.com/charmbracelet/lipgloss"

    "the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) View() string {
    if m.application.Current == StateAuthentication {
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
        layout.InputStyle.Render("> "+m.ui.Input),
        "",
        layout.StatusStyle.Render(m.communication.StatusMessage),
    )

    return layout.PanelStyle.Render(content)
}

func (m Model) renderView() string {
	result := m.calculateLayout()

	if !result.MainView.Visible {
		return layout.WarningStyle.Render("Terminal too small")
	}

	// All content panels share the same outer width.
	contentWidth := result.MainView.Width + result.Sidebar.Width

	header := layout.RenderHeader(
		m.player.Username,
		m.player.Status.HP,
		m.player.Status.MaxHP,
		contentWidth,
	)

	sidebarContent := ""
	if result.Sidebar.Visible {
		sidebarContent = m.ui.SidebarViewport.View()
	}

	body := layout.RenderMain(
		m.ui.MainViewport.View(),
		sidebarContent,
		result.MainView.Width,
		result.Sidebar.Width,
		result.MainView.Height,
		result.Sidebar.Visible,
	)

	sections := []string{
		header,
		body,
	}

	// Event Log and Chat share the same available width as Main View and Sidebar.
	if result.EventLog.Visible || result.Chat.Visible {
		var bottomPanels []string

		if result.EventLog.Visible {
			bottomPanels = append(
				bottomPanels,
				m.renderEventLog(
					result.EventLog.Width,
					result.EventLog.Height,
				),
			)
		}

		if result.Chat.Visible {
			bottomPanels = append(
				bottomPanels,
				m.renderChat(
					result.Chat.Width,
					result.Chat.Height,
				),
			)
		}

		sections = append(
			sections,
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				bottomPanels...,
			),
		)
	}

	// Available Commands uses the same outer width.
	if result.AvailableCommands.Visible {
		sections = append(
			sections,
			m.renderAvailableCommands(contentWidth),
		)
	}

	// Input and Status remain outside the content panels.
	input := layout.InputStyle.
		Width(contentWidth).
		Render("> " + m.ui.Input)

	status := layout.StatusStyle.
		Width(contentWidth).
		Render(m.communication.StatusMessage)

	sections = append(sections, input, status)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		sections...,
	)
}

func (m Model) renderWorld() string {
    return renderRoom(
        m.world.Room,
        m.ui.MainViewport.Width,
        m.ui.MainViewport.Height,
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

func (m Model) eventLogContent() string {
	var content strings.Builder

	content.WriteString(layout.TitleStyle.Render("EVENT LOG"))
	content.WriteString("\n")

	for _, event := range m.communication.EventLog {
		content.WriteString(event)
		content.WriteString("\n")
	}

	return content.String()
}

func (m Model) renderEventLog(width, height int) string {
	visibleEvents := maxInt(
		0,
		height-layout.PanelHeightOffset-1,
	)

	events := m.communication.EventLog
	start := maxInt(0, len(events)-visibleEvents)

	var content strings.Builder

	content.WriteString(layout.TitleStyle.Render("EVENT LOG"))
	content.WriteString("\n")

	for _, event := range events[start:] {
		content.WriteString(event)
		content.WriteString("\n")
	}

	contentWidth := maxInt(0, width-layout.PanelWidthOffset)
	panelHeight := maxInt(0, height-layout.PanelHeightOffset)

	return layout.PanelStyle.
		Width(contentWidth).
		Height(panelHeight).
		Render(content.String())
}

func (m Model) renderAvailableCommands(width int) string {
	contentWidth := maxInt(0, width-layout.PanelWidthOffset)

	return layout.PanelStyle.
		Width(contentWidth).
		Render(m.availableCommandsContent())
}