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
	header := layout.RenderHeader(
        m.username,
        m.playerStatus.HP,
        m.playerStatus.MaxHP,
        m.width,
    )

	input := layout.InputStyle.Render("> " + m.input)
	status := layout.StatusStyle.Render(m.status)

	// El header, el input y el status se mantienen visibles
	// incluso cuando el terminal es demasiado pequeño.
	if m.terminalTooSmall {
		warning := layout.WarningStyle.Render("Terminal too small")

		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			warning,
			input,
			status,
		)
	}

	// Main View y Sidebar comparten la misma altura exterior.
	bodyHeight := m.mainViewport.Height +
		layout.PanelHeightOffset

	mainWidth := m.width
	if m.showSidebar {
		mainWidth = m.width * 75 / 100
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

	sections := []string{
		header,
		body,
	}

	// Los bottom panels solo aparecen si hay espacio suficiente.
	if m.showBottomPanels {
		leftWidth := m.width / 2
		rightWidth := m.width - leftWidth

		eventLog := m.renderEventLog(leftWidth)
		chat := m.renderChat(rightWidth)

		bottomPanels := lipgloss.JoinHorizontal(
			lipgloss.Top,
			eventLog,
			chat,
		)

		sections = append(sections, bottomPanels)
	}

	// Available commands es la sección de menor prioridad.
	if m.showAvailableCommands {
		availableCommands := m.renderAvailableCommands(mainWidth)
		sections = append(sections, availableCommands)
	}

	sections = append(sections, input, status)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		sections...,
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

func (m Model) renderAvailableCommands(width int) string {
    commands := make([]AvailableCommand, 0)

    addCommand := func(name string) {
        for _, command := range availableCommands {
            if command.Name == name {
                commands = append(commands, command)
                return
            }
        }
    }

    addCommand("LOOK")

    if m.hasAvailableExits() {
        addCommand("MOVE")
    }

    if m.hasRoomItems() {
        addCommand("TAKE")
    }

    addCommand("INVENTORY")

    if m.hasFriendlyNPC() {
        addCommand("TALK")
    }

    addCommand("QUESTS")
    addCommand("WHO")

    if m.hasGroup() {
        commands = append(commands, groupCommands[1]) // INVITE
        commands = append(commands, groupCommands[5]) // INFO
        commands = append(commands, groupCommands[4]) // LEAVE
    } else {
        commands = append(commands, groupCommands[0]) // CREATE
    }

    addCommand("CHAT")
    addCommand("STATUS")

    if !m.inCombat && m.hasHostileNPC() {
        addCommand("ATTACK")
    }

    if m.inCombat {
        commands = append(commands, combatCommands...)
    }

    var content strings.Builder

    content.WriteString(layout.TitleStyle.Render("AVAILABLE COMMANDS"))
    content.WriteString("\n")

    usages := make([]string, 0, len(commands))

    for _, command := range commands {
        usages = append(usages, command.Usage)
    }

    content.WriteString(strings.Join(usages, "  |  "))

    contentWidth := width - layout.PanelWidthOffset
    if contentWidth < 0 {
        contentWidth = 0
    }

    return layout.PanelStyle.
        Width(contentWidth).
        Render(content.String())
}