
package ui

import (
    "strings"

    tea "github.com/charmbracelet/bubbletea"

    "the-answer-protocol/frontend/cli/ui/layout"
)

const bottomPanelHeight = layout.PanelHeightOffset + 4

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if isScrollKey(msg.String()) {
            return m.handleKey(msg)
        }

        return refreshResult(m.handleKey(msg))

    case commandResultMsg:
        return refreshResult(m.handleCommandResult(msg))

    case authenticationResultMsg:
        return refreshResult(m.handleAuthenticationResult(msg))

    case serverEventMsg:
        return refreshResult(m.handleServerEvent(msg))

    case clientDoneMsg:
        m.communication.StatusMessage = "Connection to server closed"
        return refreshResult(m, nil)

    case tea.WindowSizeMsg:
        m.ui.Width = msg.Width
        m.ui.Height = msg.Height
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

        command := m.ui.Input
        m.ui.Input = ""

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

    case "f1":
        m.navigation.CurrentView = ViewWorld
    case "f2":
        m.navigation.CurrentView = ViewRoom
    case "f3":
        m.navigation.CurrentView = ViewInventory
    case "f4":
        m.navigation.CurrentView = ViewCharacter
    case "f5":
        m.navigation.CurrentView = ViewQuests
    case "f6":
        m.navigation.CurrentView = ViewGroup
    case "f7":
        m.navigation.CurrentView = ViewCombat

    case "up", "down", "pgup", "pgdown", "home", "end":
        var cmd tea.Cmd

        m.ui.MainViewport, cmd = m.ui.MainViewport.Update(msg)

        // Scrolling only updates the Main View viewport.
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
    input := layout.DefaultRequirements()

    input.Terminal = layout.Size{
        Width:  m.ui.Width,
        Height: m.ui.Height,
    }

    input.Header.IntrinsicSize = layout.MeasureHeader(
        m.player.Username,
        m.player.Status.HP,
        m.player.Status.MaxHP,
        m.ui.Width,
    )

    commandsContent := m.availableCommandsContent()
    input.AvailableCommands.IntrinsicSize = layout.MeasureAvailableCommands(
        commandsContent,
        m.ui.Width,
    )

    input.Status.IntrinsicSize = layout.MeasureContent(
        m.communication.StatusMessage,
        m.ui.Width,
    )

    return (layout.LayoutManager{}).Calculate(input)
}

// updateViewportSizes applies the calculated dimensions to the viewports.
func (m *Model) updateViewportSizes() {
    result := m.calculateLayout()

    if !result.MainView.Visible {
        m.ui.MainViewport.Width = 0
        m.ui.MainViewport.Height = 0
        m.ui.SidebarViewport.Width = 0
        m.ui.SidebarViewport.Height = 0
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
        content = m.renderQuests()

    case ViewGroup:
        content = m.renderGroup()

    case ViewCombat:
        content = m.renderCombat()

    default:
        content = "Unknown view"
    }

    m.ui.MainViewport.SetContent(content)

    sidebar := layout.RenderSidebar(
        m.renderQuests(),
        m.renderGroup(),
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