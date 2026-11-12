package ui

import tea "github.com/charmbracelet/bubbletea"

const maxEventLogEntries = 100

func (m Model) handleServerEvent(
    msg serverEventMsg,
) (tea.Model, tea.Cmd) {
    m.eventLog = append(m.eventLog, msg.event.Raw)

    // Conservamos únicamente los últimos 100 eventos.
    if len(m.eventLog) > maxEventLogEntries {
        m.eventLog = m.eventLog[len(m.eventLog)-maxEventLogEntries:]
    }

    return m, waitForServerEvent(m.client)
}