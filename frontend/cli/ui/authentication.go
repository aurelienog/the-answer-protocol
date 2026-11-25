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
    m.player.Status.HP = 100
    m.player.Status.MaxHP = 100
    m.application.Current = StateGame
    m.world.Room.Room.ID = "loc.central_harbor_square"
    m.world.Room.Room.Name = "Central Harbor Square"
    m.communication.StatusMessage = "Successfully connected"

    return m, tea.Batch(
        waitForServerEvent(m.client),
    )
}
