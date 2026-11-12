package ui

import (
    "strings"

    tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return m.handleKey(msg)

    case commandResultMsg:
        return m.handleCommandResult(msg)
   

	case authenticationResultMsg:
    	return m.handleAuthenticationResult(msg)

    case serverEventMsg:
        return m.handleServerEvent(msg)

    case clientDoneMsg:
        m.status = "Connection to server closed"
        return m, nil

    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height

    }
    
    return m, nil
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
    default:
        if len(msg.Runes) > 0 {
            m.input += string(msg.Runes)
        }
    }

    return m, nil
}

