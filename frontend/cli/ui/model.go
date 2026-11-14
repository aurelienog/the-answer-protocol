
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


type ApplicationState struct {
    Current       AppState
    AuthInProgress bool
}

type NavigationState struct {
    CurrentView View
}

type PlayerState struct {
    Username string
    Status   PlayerStatus
}

type WorldState struct {
    Room     LookResponse
    Quests   []Quest
    Group    Group
    InCombat bool
    CombatID string
}

type CommunicationState struct {
    StatusMessage string
    EventLog      []string
    ChatLog       []ChatMessage
    ChatFilter    ChatFilter
}

type InterfaceState struct {
    Input  string
    Width  int
    Height int

    MainViewport    viewport.Model
    SidebarViewport viewport.Model
}

type Model struct {
    client *network.Client

    application    ApplicationState
    navigation    NavigationState
    player         PlayerState
    world          WorldState
    communication CommunicationState
    ui             InterfaceState
}

func NewModel(client *network.Client) Model {
    return Model{
        client: client,

        application: ApplicationState{
            Current: StateAuthentication,
        },

        navigation: NavigationState{
            CurrentView: ViewWorld,
        },

        communication: CommunicationState{
            StatusMessage: "Enter your username",
            ChatFilter:    ChatFilterAll,
        },

        ui: InterfaceState{
            MainViewport:    viewport.New(0, 0),
            SidebarViewport: viewport.New(0, 0),
        },
    }
}

func (m Model) Init() tea.Cmd {
	return nil
}
