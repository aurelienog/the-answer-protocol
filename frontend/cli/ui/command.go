package ui

import (
    "context"

    tea "github.com/charmbracelet/bubbletea"
    "the-answer-protocol/frontend/cli/network"
)

func sendCommand(client *network.Client, command string) tea.Cmd {
    return func() tea.Msg {
        response, err := client.Send(
            context.Background(),
            command,
        )

        return commandResultMsg{
            command:  command,
            response: response,
            err:      err,
        }
    }
}

func authenticateUser(client *network.Client, username string) tea.Cmd {
    return func() tea.Msg {
        err := client.Authenticate(username)

        return authenticationResultMsg{
            username: username,
            err:      err,
        }
    }
}