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
}

var commandErrorHandlers = map[string]errorHandler{
    "NOT_IN_GROUP": func(m *Model) {
        m.group = Group{}
    },
}

func (m Model) handleCommandResult(msg commandResultMsg) (tea.Model, tea.Cmd) {
    if msg.err != nil {
        m.status = "Error: " + msg.err.Error()
        return m, nil
    }

    command := strings.ToUpper(strings.TrimSpace(msg.command))

    if msg.response.Kind == protocol.KindERR {
        errorCode := strings.ToUpper(strings.TrimSpace(msg.response.Name))
        m.status = errorCode

        if handler, exists := commandErrorHandlers[errorCode]; exists {
            handler(&m)
        }

        return m, nil
    }

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

    m.status = command + " updated"
    return m, nil
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