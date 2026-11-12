package main

import (
    "context"
    "log"

    tea "github.com/charmbracelet/bubbletea"

    "the-answer-protocol/frontend/cli/network"
    "the-answer-protocol/frontend/cli/ui"
    "the-answer-protocol/internal/config"
)

func main() {
    cfg := config.LoadConfig()

    ctx := context.Background()

    connection, err := network.Dial(ctx, cfg.ServerAddr())
    if err != nil {
        log.Printf("Connection error: %v", err)
        return
    }

    client, err := network.NewClient(ctx, connection)
    if err != nil {
        log.Printf("Client initialization error: %v", err)
        return
    }
    defer client.Close()

    model := ui.NewModel(client)

    program := tea.NewProgram(
        model,
        tea.WithAltScreen(),
    )

    if _, err := program.Run(); err != nil {
        log.Printf("UI error: %v", err)
    }
}