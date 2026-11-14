
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

    header := layout.RenderHeader(
        m.player.Username,
        m.player.Status.HP,
        m.player.Status.MaxHP,
        m.ui.Width,
    )

    input := layout.InputStyle.Render("> " + m.ui.Input)
    status := layout.StatusStyle.Render(m.communication.StatusMessage)

    // Keep the fixed components visible when the terminal is too small.
    if !result.MainView.Visible {
        warning := layout.WarningStyle.Render("Terminal too small")

        return lipgloss.JoinVertical(
            lipgloss.Left,
            header,
            warning,
            input,
            status,
        )
    }

    bodyWidth := result.MainView.Width
    if result.Sidebar.Visible {
        bodyWidth += result.Sidebar.Width
    }

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
    )

    sections := []string{
        header,
        body,
    }

    // EventLog and Chat share a horizontal row.
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

    // AvailableCommands is displayed only when space permits.
    if result.AvailableCommands.Visible {
        sections = append(
            sections,
            m.renderAvailableCommands(result.AvailableCommands.Width),
        )
    }

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

func (m Model) renderEventLog(width, height int) string {
    title := layout.TitleStyle.Render("EVENT LOG")

    var content strings.Builder
    content.WriteString(title)
    content.WriteString("\n")

    visibleEvents := height - layout.PanelHeightOffset - 1

    if visibleEvents < 0 {
        visibleEvents = 0
    }

    start := 0
    if len(m.communication.EventLog) > visibleEvents {
        start = len(m.communication.EventLog) - visibleEvents
    }

    for _, event := range m.communication.EventLog[start:] {
        content.WriteString(event)
        content.WriteString("\n")
    }

    contentWidth := width - layout.PanelWidthOffset
    if contentWidth < 0 {
        contentWidth = 0
    }

    panelHeight := height - layout.PanelHeightOffset
    if panelHeight < 0 {
        panelHeight = 0
    }

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