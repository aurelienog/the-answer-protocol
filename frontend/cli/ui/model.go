package ui

import (
    tea "github.com/charmbracelet/bubbletea"
    "the-answer-protocol/frontend/cli/network"
)

type View uint8
type AppState uint8

const (
    StateAuthentication AppState = iota
    StateGame
    ViewWorld View = iota
    ViewRoom
    ViewInventory
    ViewCharacter
    ViewQuests
    ViewGroup
    ViewCombat
)

type Model struct {
    client *network.Client

    appState    AppState
    currentView View

    input    string
    username string
    playerStatus PlayerStatus

    status   string

    eventLog []string

    group  Group
    quests []Quest

    authInProgress bool
}

func NewModel(client *network.Client) Model {
    return Model{
        client:      client,
        appState:    StateAuthentication,
        currentView: ViewWorld,
        status:      "Enter your username",
    }
}

func (m Model) Init() tea.Cmd {
    return nil
}