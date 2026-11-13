package ui

import (
    "encoding/json"
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "the-answer-protocol/internal/protocol"
)

type commandHandler func(*Model, string) error
type errorHandler func(*Model)

var commandHandlers = map[string]commandHandler{
    "STATUS":     handleStatusResponse,
    "QUESTS":     handleQuestsResponse,
    "GROUP INFO": handleGroupInfoResponse,
    "LOOK":       handleLookResponse,
    "MOVE":       handleMoveResponse,
}

var commandErrorHandlers = map[string]errorHandler{
    "NOT_IN_GROUP": func(m *Model) {
        m.group = Group{}
    },
}

func (m Model) handleCommandResult(msg commandResultMsg) (tea.Model, tea.Cmd) {
    // Handle server error responses first.
    if msg.response.Kind == protocol.KindERR {
        errorCode := strings.ToUpper(strings.TrimSpace(msg.response.Name))
        m.status = errorCode

        if handler, exists := commandErrorHandlers[errorCode]; exists {
            handler(&m)
        }

        return m, nil
    }

    // Handle transport and client errors.
    if msg.err != nil {
        m.status = "Error: " + msg.err.Error()
        return m, nil
    }

    command := getCommandName(msg.command)

    if msg.response.Kind != protocol.KindOK {
        m.status = "Unexpected server response"
        return m, nil
    }

    handler, exists := commandHandlers[command]
    if !exists {
        m.status = "Command executed"
        return m, nil
    }

    if err := handler(&m, msg.response.Payload); err != nil {
        m.status = "Error parsing " + command + ": " + err.Error()
        return m, nil
    }

    if command == "MOVE" {
        m.status = "Movement successful. Updating room..."
        return m, sendCommand(m.client, "LOOK")
    }

    m.status = command + " updated"
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

func handleStatusResponse(m *Model, payload string) error {
    var status PlayerStatus

    if err := json.Unmarshal([]byte(payload), &status); err != nil {
        return err
    }

    m.playerStatus = status
    return nil
}

func handleQuestsResponse(m *Model, payload string) error {
    var quests []Quest

    if err := json.Unmarshal([]byte(payload), &quests); err != nil {
        return err
    }

    m.quests = quests
    return nil
}

func handleGroupInfoResponse(m *Model, payload string) error {
    var group Group

    if err := json.Unmarshal([]byte(payload), &group); err != nil {
        return err
    }

    m.group = group
    return nil
}

func handleLookResponse(m *Model, payload string) error {
    var response LookResponse

    if err := json.Unmarshal([]byte(payload), &response); err != nil {
        return err
    }

    m.currentRoom = response
    return nil
}

func handleMoveResponse(m *Model, payload string) error {
    return nil
}