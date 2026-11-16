package ui

import (
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "the-answer-protocol/internal/protocol"
)

type commandHandler func(*Model, string) error
type errorHandler func(*Model)

var commandHandlers = map[string]commandHandler{
    "STATUS":     handleStatusResponse,
    "QUESTS":     handleQuestsResponse,
    "QUEST":      handleQuestResponse,
    "GROUP INFO": handleGroupInfoResponse,
    "LOOK":       handleLookResponse,
    "MOVE":       handleMoveResponse,
    "TALK":       handleTalkResponse,
    "INVENTORY":  handleInventoryResponse,
    "TAKE":       handleTakeResponse,
    "DROP":       handleDropResponse,
}

var commandErrorHandlers = map[string]errorHandler{
    "NOT_IN_GROUP": func(m *Model) {
        m.world.Group = Group{}
    },
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

	if err := handler(&m, msg.response.Payload); err != nil {
		m.ui.PendingRefresh = RefreshNone
		m.communication.StatusMessage = "Error parsing " + command + ": " + err.Error()
		return m, nil
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