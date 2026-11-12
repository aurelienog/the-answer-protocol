package ui

import (
    tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleAuthenticationResult(
    msg authenticationResultMsg,
) (tea.Model, tea.Cmd) {
    m.authInProgress = false

    if msg.err != nil {
        m.status = msg.err.Error()
        return m, nil
    }

    m.username = msg.username
    m.appState = StateGame
    m.status = "Successfully connected"

    return m, tea.Batch(
        sendCommand(m.client, "STATUS"),
        waitForServerEvent(m.client),
    )
}