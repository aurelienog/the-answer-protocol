package ui

import (
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "the-answer-protocol/internal/protocol"
)

type commandHandler func(*Model, string) error
type errorHandler func(*Model)

var commandHandlers = map[string]commandHandler{
    "STATUS":       handleStatusResponse,
    "QUESTS":       handleQuestsResponse,
    "QUEST":        handleQuestResponse,
    "GROUP INFO":   handleGroupInfoResponse,
    "GROUP CREATE": handleGroupActionResponse,
    "GROUP JOIN":   handleGroupActionResponse,
    "GROUP INVITE": handleGroupActionResponse,
    "GROUP LEAVE":  handleGroupActionResponse,
    "LOOK":         handleLookResponse,
    "MOVE":         handleMoveResponse,
    "TALK":         handleTalkResponse,
    "INVENTORY":    handleInventoryResponse,
    "TAKE":         handleTakeResponse,
    "DROP":         handleDropResponse,
}

var commandErrorHandlers = map[string]errorHandler{
    "NOT_IN_GROUP": func(m *Model) {
        m.world.Group = Group{}
    },
}

func handleGroupActionResponse(_ *Model, _ string) error {
    return nil
}


func (m Model) handleCommandResult(msg commandResultMsg) (tea.Model, tea.Cmd) {
	// Handle transport and client errors first.
	if msg.err != nil {
		m.ui.PendingRefresh = RefreshNone
		m.communication.StatusMessage = "Error: " + msg.err.Error()
		return m, nil
	}

	// Handle server error responses.
	if msg.response.Kind == protocol.KindERR {
		m.ui.PendingRefresh = RefreshNone

		errorCode := strings.ToUpper(strings.TrimSpace(msg.response.Name))
		m.communication.StatusMessage = errorCode

		if handler, exists := commandErrorHandlers[errorCode]; exists {
			handler(&m)
		}

		return m, nil
	}

	command := getCommandName(msg.command)

	if msg.response.Kind != protocol.KindOK {
		m.ui.PendingRefresh = RefreshNone
		m.communication.StatusMessage = "Unexpected server response"
		return m, nil
	}

	handler, exists := commandHandlers[command]
	if !exists {
		m.communication.StatusMessage = "Command executed"
		return m, nil
	}

	pendingRefresh := m.ui.PendingRefresh

	if err := handler(&m, msg.response.Payload); err != nil {
		m.ui.PendingRefresh = RefreshNone
		m.communication.StatusMessage = "Error parsing " + command + ": " + err.Error()
		return m, nil
	}

	// Handle commands that navigate to another view.
	switch command {
	case "LOOK":
		if pendingRefresh != RefreshRoom {
			return m, m.startViewTransition(ViewRoom)
		}

	case "INVENTORY":
		if pendingRefresh != RefreshInventory {
			return m, m.startViewTransition(ViewInventory)
		}

	case "QUESTS":
		if pendingRefresh != RefreshQuests {
			return m, m.startViewTransition(ViewQuests)
		}

	case "TALK":
		return m, m.startViewTransition(ViewTalk)

	case "GROUP INFO":
		if pendingRefresh != RefreshGroup {
			return m, m.startViewTransition(ViewGroup)
		}
	}

	if command == "QUEST" {
		m.communication.StatusMessage = "Quest accepted. Updating quests..."
		m.ui.PendingRefresh = RefreshQuests
		return m, sendCommand(m.client, "QUESTS")
	}

	if command == "DROP" || command == "TAKE" {
		m.communication.StatusMessage = "Item updated. Refreshing data..."

		switch m.navigation.CurrentView {
		case ViewInventory:
			m.ui.PendingRefresh = RefreshInventory
			return m, sendCommand(m.client, "INVENTORY")

		case ViewRoom, ViewWorld:
			m.ui.PendingRefresh = RefreshRoom
			return m, sendCommand(m.client, "LOOK")
		}
	}
	// Refresh group information after group operations.
	switch command {
	case "GROUP CREATE", "GROUP JOIN":
		m.communication.StatusMessage = "Group updated. Loading group..."
		m.ui.PendingRefresh = RefreshNone
		return m, sendCommand(m.client, "GROUP INFO")

	case "GROUP INVITE":
		m.communication.StatusMessage = "Invitation sent."
		m.ui.PendingRefresh = RefreshGroup
		return m, sendCommand(m.client, "GROUP INFO")

	case "GROUP LEAVE":
		m.world.Group = Group{}
		m.communication.StatusMessage = "You left the group."
		return m, nil
	}

	if command == "MOVE" {
		m.communication.StatusMessage = "Movement successful. Updating room..."
		return m, sendCommand(m.client, "LOOK")
	}

	if command == "QUIT" {
		return m, tea.Quit
	}

	m.communication.StatusMessage = command + " updated"
	return m, nil
}

func getCommandName(rawCommand string) string {
    parts := strings.Fields(strings.ToUpper(rawCommand))

    if len(parts) == 0 {
        return ""
    }

    // Algunos comandos tienen dos palabras, como GROUP INFO.
    if len(parts) >= 2 {
        twoWords := parts[0] + " " + parts[1]
        if _, exists := commandHandlers[twoWords]; exists {
            return twoWords
        }
    }

    return parts[0]
}