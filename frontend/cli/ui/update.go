package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"the-answer-protocol/frontend/cli/ui/layout"
	"the-answer-protocol/frontend/cli/ui/animations"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if isScrollKey(msg.String()) {
			return m.handleKey(msg)
		}

		return refreshResult(m.handleKey(msg))

	case commandResultMsg:
		model, cmd := m.handleCommandResult(msg)

		updated, ok := model.(Model)
		if !ok {
			return model, cmd
		}

		m = updated

		if m.syncCombatHealthBars() {
			cmd = tea.Batch(cmd, healthBarTick())
		}

		return refreshResult(m, cmd)

	case authenticationResultMsg:
		return refreshResult(m.handleAuthenticationResult(msg))

	case serverEventMsg:
		model, cmd := m.handleServerEvent(msg)

		updated, ok := model.(Model)
		if !ok {
			return model, cmd
		}

		m = updated

		var batch []tea.Cmd
		if cmd != nil {
			batch = append(batch, cmd)
		}
		if m.client != nil {
			batch = append(batch, waitForServerEvent(m.client))
		}
		if m.syncCombatHealthBars() {
			batch = append(batch, healthBarTick())
		}

		return refreshResult(m, tea.Batch(batch...))

	case clientDoneMsg:
		m.communication.StatusMessage = "Connection to server closed"
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.ui.Width = msg.Width
		m.ui.Height = msg.Height
	
	case animations.CursorBlinkMsg:
		m.ui.CursorVisible = !m.ui.CursorVisible
		return refreshResult(m, animations.CursorBlink())

	case healthBarTickMsg:
		animating := false

		for id, bar := range m.ui.healthBars {
			if bar.Tick() {
				animating = true
			}

			// Persistimos el estado actualizado.
			m.ui.healthBars[id] = bar
		}

		if animating {
			return refreshResult(m, healthBarTick())
		}

		return refreshResult(m, nil)
	
	case TransitionMsg:
		switch msg.Phase {
		case TransitionReveal:
			m.ui.TransitionPhase = TransitionReveal
			return m, NextTransition(TransitionComplete)

		case TransitionComplete:
			m.setView(m.ui.PendingView)
			m.ui.Transitioning = false
			m.ui.TransitionPhase = 0
			m.ui.PendingView = 0

			return refreshResult(m, nil)
		}

	}
	return refreshResult(m, nil)
}

// Enter submits the username during authentication or a game command afterward.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "enter":
		if strings.TrimSpace(m.ui.Input) == "" {
			return m, nil
		}

		if m.application.Current == StateAuthentication {
			if m.application.AuthInProgress {
				return m, nil
			}

			username := strings.TrimSpace(m.ui.Input)

			m.ui.Input = ""
			m.application.AuthInProgress = true
			m.communication.StatusMessage = "Authenticating..."

			return m, authenticateUser(m.client, username)
		}

		if m.client == nil {
			m.communication.StatusMessage = "Client not connected"
			m.ui.Input = ""
			return m, nil
		}

		command := strings.TrimSpace(m.ui.Input)
		m.ui.Input = ""
		m.ui.InputOffset = 0

		if strings.EqualFold(command, "MAP") {
			m.communication.StatusMessage = "Returning to world view..."
			return m, m.startViewTransition(ViewWorld)
		}

		// Combat navigation: "combat" opens the view without sending a command.
		if strings.EqualFold(command, "combat") {
			if m.navigation.CurrentView != ViewCombat {
				return m, m.startViewTransition(ViewCombat)
			}

			return m, nil
		}

		// Combat actions: open the view with transition and send the action to the server.
		if strings.HasPrefix(strings.ToLower(command), "combat ") {
			sendActionCmd := sendCommand(m.client, command)

			if m.navigation.CurrentView != ViewCombat {
				return m, tea.Batch(
					m.startViewTransition(ViewCombat),
					sendActionCmd,
				)
			}

			return m, sendActionCmd
		}

		return m, sendCommand(m.client, command)

	case "backspace":
		if len(m.ui.Input) > 0 {
			m.ui.Input = m.ui.Input[:len(m.ui.Input)-1]
		}
		return m, nil

	case "tab":
		m.communication.ChatFilter =
			(m.communication.ChatFilter + 1) % 4
		return m, nil

	case "<":
		result := m.calculateLayout()

		if result.Sidebar.Visible {
			m.ui.PanelFocused = !m.ui.PanelFocused
		} else {
			m.ui.PanelFocused = false
		}

		return m, nil

	case "up", "down", "pgup", "pgdown", "home", "end":
		var cmd tea.Cmd

		if m.ui.PanelFocused {
			m.ui.SidebarViewport, cmd = m.ui.SidebarViewport.Update(msg)
		} else {
			m.ui.MainViewport, cmd = m.ui.MainViewport.Update(msg)
		}

		return m, cmd

	default:
		if len(msg.Runes) > 0 {
			m.ui.Input += string(msg.Runes)
		}
	}

	return m, nil
}

// calculateLayout measures the components and returns their layout.
func (m Model) calculateLayout() layout.LayoutResult {
	manager := layout.LayoutManager{}
	input := layout.DefaultRequirements()

	input.Terminal = layout.Size{
		Width:  m.ui.Width,
		Height: m.ui.Height,
	}

	// First, distribute horizontal space.
	horizontal := manager.CalculateHorizontal(input)

	contentWidth := horizontal.MainView.Width + horizontal.Sidebar.Width

	// Measure components using their available widths.
	input.Header.IntrinsicSize = layout.MeasureHeader(
		m.player.Username,
		m.player.Status.HP,
		m.player.Status.MaxHP,
		contentWidth,
	)

	input.AvailableCommands.IntrinsicSize = layout.MeasureAvailableCommands(
		m.availableCommandsContent(),
		horizontal.AvailableCommands.Width,
	)

	input.Status.IntrinsicSize = layout.MeasureContent(
		m.communication.StatusMessage,
		horizontal.Status.Width,
	)

	return manager.Calculate(input, horizontal)
}

// updateViewportSizes applies the calculated dimensions to the viewports.
func (m *Model) updateViewportSizes() {
	result := m.calculateLayout()

	if !result.MainView.Visible {
		m.ui.MainViewport.Width = 0
		m.ui.MainViewport.Height = 0
		m.ui.SidebarViewport.Width = 0
		m.ui.SidebarViewport.Height = 0
		m.ui.PanelFocused = false
		return
	}

	viewportHeight := maxInt(
		0,
		result.MainView.Height-layout.PanelHeightOffset,
	)

	m.ui.MainViewport.Width = maxInt(
		0,
		result.MainView.Width-layout.PanelWidthOffset,
	)
	m.ui.MainViewport.Height = viewportHeight

	if !result.Sidebar.Visible {
		m.ui.SidebarViewport.Width = 0
		m.ui.SidebarViewport.Height = 0
		m.ui.PanelFocused = false
		return
	}

	m.ui.SidebarViewport.Width = maxInt(
		0,
		result.Sidebar.Width-layout.PanelWidthOffset,
	)
	m.ui.SidebarViewport.Height = viewportHeight
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func (m *Model) refreshViewports() {
	var content string

	switch m.navigation.CurrentView {
	case ViewWorld:
		content = m.renderWorld()

	case ViewRoom:
		content = renderRoom(
			m.world.Room,
			m.ui.MainViewport.Width,
			m.ui.MainViewport.Height,
		)

	case ViewInventory:
		content = m.renderInventory()

	case ViewCharacter:
		content = m.renderCharacter()

	case ViewQuests:
		content = m.renderQuests(true, m.ui.MainViewport.Width)

	case ViewGroup:
		content = m.renderGroup(true)

	case ViewCombat:
		content = m.renderCombat()

	case ViewTalk:
    	content = m.renderTalk()
	default:
		content = "Unknown view"
	}

	m.ui.MainViewport.SetContent(content)

	sidebar := layout.RenderSidebar(
		m.renderQuests(false, m.ui.SidebarViewport.Width),
		m.renderGroup(false),
		m.ui.SidebarViewport.Width,
	)

	m.ui.SidebarViewport.SetContent(sidebar)
}

func refreshResult(model tea.Model, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	if m, ok := model.(Model); ok {
		m.updateViewportSizes()
		m.refreshViewports()

		return m, cmd
	}

	return model, cmd
}

func isScrollKey(key string) bool {
	switch key {
	case "up", "down", "pgup", "pgdown", "home", "end":
		return true
	default:
		return false
	}
}
