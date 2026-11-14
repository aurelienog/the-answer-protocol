package ui

import (
    tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleAuthenticationResult(
    msg authenticationResultMsg,
) (tea.Model, tea.Cmd) {
    m.application.AuthInProgress = false

    if msg.err != nil {
        m.communication.StatusMessage = msg.err.Error()
        return m, nil
    }

    m.player.Username = msg.username
    m.application.Current = StateGame
    m.communication.StatusMessage = "Successfully connected"

    return m, tea.Batch(
        sendCommand(m.client, "STATUS"),
        sendCommand(m.client, "LOOK"),
        waitForServerEvent(m.client),
    )
}
