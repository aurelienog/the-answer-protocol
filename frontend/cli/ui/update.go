package ui

import (
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"

    "the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return refreshResult(m.handleKey(msg))

	case commandResultMsg:
		return refreshResult(m.handleCommandResult(msg))

	case authenticationResultMsg:
		return refreshResult(m.handleAuthenticationResult(msg))

	case serverEventMsg:
		return refreshResult(m.handleServerEvent(msg))

	case clientDoneMsg:
		m.status = "Connection to server closed"
		return refreshResult(m, nil)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateViewportSizes()
	}

	return refreshResult(m, nil)
}

// Durante la autenticación, Enter envía el nombre de usuario.
// Después de autenticarse, Enter envía comandos del juego.
// No necesitamos dos sistemas diferentes de entrada.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "ctrl+c":
        return m, tea.Quit

    case "enter":
        if strings.TrimSpace(m.input) == "" {
            return m, nil
        }

        if m.appState == StateAuthentication {
            if m.authInProgress {
                return m, nil
            }

            username := strings.TrimSpace(m.input)

            m.input = ""
            m.authInProgress = true
            m.status = "Authenticating..."

            return m, authenticateUser(m.client, username)
        }

        if m.client == nil {
            m.status = "Client not connected"
            m.input = ""
            return m, nil
        }

        command := m.input
        m.input = ""

        return m, sendCommand(m.client, command)

    case "backspace":
        if len(m.input) > 0 {
            m.input = m.input[:len(m.input)-1]
        }
        return m, nil

	case "tab":
		m.chatFilter = (m.chatFilter + 1) % 4
		return m, nil

    case "f1":
        m.currentView = ViewWorld
    case "f2":
        m.currentView = ViewRoom
    case "f3":
        m.currentView = ViewInventory
    case "f4":
        m.currentView = ViewCharacter
    case "f5":
        m.currentView = ViewQuests
    case "f6":
        m.currentView = ViewGroup
    case "f7":
        m.currentView = ViewCombat
    case "up", "down", "pgup", "pgdown", "home", "end":
        var cmd tea.Cmd
        m.mainViewport, cmd = m.mainViewport.Update(msg)
        return m, cmd
    default:
        if len(msg.Runes) > 0 {
            m.input += string(msg.Runes)
        }
    }

    return m, nil
}

func (m *Model) updateViewportSizes() {
	header := layout.RenderHeader(
		m.username,
		m.playerStatus.HP,
		m.playerStatus.MaxHP,
	)

	inputHeight := lipgloss.Height(
		layout.InputStyle.Render("> " + m.input),
	)
	statusHeight := lipgloss.Height(
		layout.StatusStyle.Render(m.status),
	)

	fixedHeight := lipgloss.Height(header) +
		inputHeight +
		statusHeight

	minMainHeight := layout.PanelHeightOffset +
		layout.MinMainHeight

	minEventLogHeight := layout.PanelHeightOffset +
		1 +
		layout.EventLogMinEntries

	availableHeight := m.height - fixedHeight

	// Comprobar si cabe el contenido obligatorio.
	m.terminalTooSmall = m.width < layout.MinTerminalWidth ||
		availableHeight < minMainHeight+minEventLogHeight

	if m.terminalTooSmall {
		m.mainViewport.Width = 0
		m.mainViewport.Height = 0
		m.sidebarViewport.Width = 0
		m.sidebarViewport.Height = 0
		m.showSidebar = false
		return
	}

	// El event log crece con el espacio sobrante.
	m.eventLogHeight = minInt(
		layout.EventLogMaxHeight,
		availableHeight-minMainHeight,
	)

	viewportHeight := availableHeight -
		m.eventLogHeight -
		layout.PanelHeightOffset

	m.showSidebar = m.width >= layout.MinWidthForSidebar &&
		viewportHeight >= layout.MinSidebarHeight

	mainWidth := m.width

	if m.showSidebar {
		mainWidth = m.width * 75 / 100
		sidebarWidth := m.width - mainWidth

		m.sidebarViewport.Width = maxInt(
			0,
			sidebarWidth-layout.PanelWidthOffset,
		)
	} else {
		m.sidebarViewport.Width = 0
	}

	m.mainViewport.Width = maxInt(
		0,
		mainWidth-layout.PanelWidthOffset,
	)
	m.mainViewport.Height = viewportHeight
	m.sidebarViewport.Height = viewportHeight
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
    if a > b {
        return a
    }
    return b
}

func (m *Model) refreshViewports() {
	var content string

	switch m.currentView {
	case ViewWorld:
        content = m.renderWorld()
    case ViewRoom:
        content = renderRoom(
            m.currentRoom,
            m.mainViewport.Width,
            m.mainViewport.Height,
        )
	case ViewInventory:
		content = m.renderInventory()
	case ViewCharacter:
		content = m.renderCharacter()
	case ViewQuests:
		content = m.renderQuests()
	case ViewGroup:
		content = m.renderGroup()
	case ViewCombat:
		content = m.renderCombat()
	default:
		content = "Unknown view"
	}

	m.mainViewport.SetContent(content)

	sidebar := layout.RenderSidebar(
		m.renderQuests(),
		m.renderGroup(),
	)
	m.sidebarViewport.SetContent(sidebar)
}

func refreshResult(model tea.Model, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	if m, ok := model.(Model); ok {
		m.refreshViewports()
		return m, cmd
	}

	return model, cmd
}