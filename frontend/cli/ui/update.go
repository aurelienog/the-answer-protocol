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
		m.width,
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

	availableHeight := m.height - fixedHeight

	minMainHeight := layout.PanelHeightOffset +
		layout.MinMainHeight

	minEventLogHeight := layout.PanelHeightOffset +
		1 +
		layout.EventLogMinEntries

	m.terminalTooSmall = m.width < layout.MinTerminalWidth ||
		availableHeight < minMainHeight

	if m.terminalTooSmall {
		m.showBottomPanels = false
		m.showAvailableCommands = false
		m.showSidebar = false
		m.eventLogHeight = 0

		m.mainViewport.Width = 0
		m.mainViewport.Height = 0
		m.sidebarViewport.Width = 0
		m.sidebarViewport.Height = 0

		return
	}

	// Espacio disponible después de reservar el mínimo del Main View.
	remainingHeight := availableHeight - minMainHeight

	// Los bottom panels tienen prioridad sobre los comandos.
	m.showBottomPanels = remainingHeight >= minEventLogHeight
	m.eventLogHeight = 0

	if m.showBottomPanels {
		m.eventLogHeight = minInt(
			layout.EventLogMaxHeight,
			remainingHeight,
		)
	}

	remainingHeight -= m.eventLogHeight

	// Calcula el ancho inicial suponiendo que la sidebar está visible.
	mainWidth := m.width
	sidebarCandidate := m.width >= layout.MinWidthForSidebar

	if sidebarCandidate {
		mainWidth = m.width * 75 / 100
	}

	// Los comandos solo aparecen si caben después de los bottom panels.
	commandHeight := lipgloss.Height(
		m.renderAvailableCommands(mainWidth),
	)

	m.showAvailableCommands = m.showBottomPanels &&
		remainingHeight >= commandHeight

	if m.showAvailableCommands {
		remainingHeight -= commandHeight
	}

	// Altura interior del Main View.
	viewportHeight := availableHeight -
		m.eventLogHeight -
		layout.PanelHeightOffset

	if m.showAvailableCommands {
		viewportHeight -= commandHeight
	}

	// La sidebar solo aparece si también dispone de altura suficiente.
	m.showSidebar = sidebarCandidate &&
		viewportHeight >= layout.MinSidebarHeight

	// Si no hay sidebar, el Main View y los comandos usan todo el ancho.
	if !m.showSidebar {
		mainWidth = m.width

		// Al cambiar el ancho, puede cambiar la altura de los comandos.
		commandHeight = lipgloss.Height(
			m.renderAvailableCommands(mainWidth),
		)

		m.showAvailableCommands = m.showBottomPanels &&
			remainingHeight+m.eventLogHeight >= commandHeight

		// Recalcula la altura con la decisión definitiva sobre los comandos.
		viewportHeight = availableHeight -
			m.eventLogHeight -
			layout.PanelHeightOffset

		if m.showAvailableCommands {
			viewportHeight -= commandHeight
		}
	}

	// La asignación anterior reserva el mínimo del Main View.
	if viewportHeight < layout.MinMainHeight {
		viewportHeight = layout.MinMainHeight
	}

	m.mainViewport.Width = maxInt(
		0,
		mainWidth-layout.PanelWidthOffset,
	)
	m.mainViewport.Height = viewportHeight

	if m.showSidebar {
		sidebarWidth := m.width - mainWidth

		m.sidebarViewport.Width = maxInt(
			0,
			sidebarWidth-layout.PanelWidthOffset,
		)
	} else {
		m.sidebarViewport.Width = 0
	}

	// Ambos viewports tienen siempre la misma altura interior.
	m.sidebarViewport.Height = m.mainViewport.Height
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
        m.updateViewportSizes()
        m.refreshViewports()

        return m, cmd
    }

    return model, cmd
}