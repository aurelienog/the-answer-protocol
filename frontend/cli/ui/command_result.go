package ui

import (
    "encoding/json"
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "the-answer-protocol/internal/protocol"
)

func (m Model) handleCommandResult(msg commandResultMsg) (tea.Model, tea.Cmd) {
    if msg.err != nil {
        m.status = "Error: " + msg.err.Error()
        return m, nil
    }

    if msg.response.Kind == protocol.KindERR {
		m.status = msg.response.Name

		if strings.ToUpper(strings.TrimSpace(msg.command)) == "GROUP INFO" &&
			msg.response.Name == "NOT_IN_GROUP" {
			m.group = Group{}
		}

		return m, nil
	}

    if msg.response.Kind != protocol.KindOK {
        m.status = "Unexpected server response"
        return m, nil
    }

    switch strings.ToUpper(strings.TrimSpace(msg.command)) {
    case "STATUS":
        var playerStatus PlayerStatus

        if err := json.Unmarshal(
            []byte(msg.response.Payload),
            &playerStatus,
        ); err != nil {
            m.status = "Error parsing player status: " + err.Error()
            return m, nil
        }

        m.playerStatus = playerStatus
        m.status = "Player status updated"
        
    case "QUESTS":
        var quests []Quest

        if err := json.Unmarshal(
            []byte(msg.response.Payload),
            &quests,
        ); err != nil {
            m.status = "Error parsing quests: " + err.Error()
            return m, nil
        }

        m.quests = quests
        m.status = "Quests updated"

    case "GROUP INFO":
        var group Group

        if err := json.Unmarshal(
            []byte(msg.response.Payload),
            &group,
        ); err != nil {
            m.status = "Error parsing group: " + err.Error()
            return m, nil
        }

        m.group = group
        m.status = "Group updated"
    }

    return m, nil
}