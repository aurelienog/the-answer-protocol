
package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"

	"the-answer-protocol/frontend/cli/network"
)

type View uint8
type AppState uint8
type ChatFilter uint8

const (
	StateAuthentication AppState = iota
	StateGame
)

const (
	ViewWorld View = iota
	ViewRoom
	ViewInventory
	ViewCharacter
	ViewQuests
	ViewGroup
	ViewCombat
)

const (
    ChatFilterAll ChatFilter = iota
    ChatFilterGlobal
    ChatFilterRoom
    ChatFilterGroup
)

type Model struct {
	client *network.Client

	appState    AppState
	currentView View
	terminalTooSmall bool
	eventLogHeight int
	showSidebar      bool

	input        string
	username     string
	playerStatus PlayerStatus

	status      string
	currentRoom LookResponse

	eventLog []string
	chatLog  []ChatMessage
	chatFilter ChatFilter

	group  Group
	quests []Quest

	authInProgress bool

	width  int
	height int

	mainViewport    viewport.Model
	sidebarViewport viewport.Model
}

func NewModel(client *network.Client) Model {
	return Model{
		client:          client,
		appState:        StateAuthentication,
		currentView:     ViewWorld,
		status:          "Enter your username",
		mainViewport:    viewport.New(0, 0),
		sidebarViewport: viewport.New(0, 0),
		chatFilter: ChatFilterAll,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}
