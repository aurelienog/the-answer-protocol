
package ui

import (
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "the-answer-protocol/internal/protocol"
)

const maxEventLogEntries = 100

func (m Model) handleServerEvent(
    msg serverEventMsg,
) (tea.Model, tea.Cmd) {
    event := msg.event.Message

    if event.Kind == protocol.KindEVT &&
        isChatEvent(event.Data) {

        chatMessage, ok := parseChatMessage(event.EventType, event.Data)
        if ok {
            m.chatLog = append(m.chatLog, chatMessage)
        }

        if len(m.chatLog) > maxEventLogEntries {
            m.chatLog = m.chatLog[len(m.chatLog)-maxEventLogEntries:]
        }
    } else {
        m.eventLog = append(m.eventLog, msg.event.Raw)

        if len(m.eventLog) > maxEventLogEntries {
            m.eventLog = m.eventLog[len(m.eventLog)-maxEventLogEntries:]
        }
    }

    return m, waitForServerEvent(m.client)
}

func isChatEvent(data string) bool {
    return strings.HasPrefix(data, "CHAT player=")
}

func formatChatMessage(scope, data string) string {
    data = strings.TrimPrefix(data, "CHAT player=")

    parts := strings.SplitN(data, " message=", 2)
    if len(parts) != 2 {
        return data
    }

    return "[" + scope + "] " + parts[0] + ": " + parts[1]
}

func parseChatMessage(scope, data string) (ChatMessage, bool) {
    data = strings.TrimPrefix(data, "CHAT player=")

    parts := strings.SplitN(data, " message=", 2)
    if len(parts) != 2 {
        return ChatMessage{}, false
    }

    return ChatMessage{
        Scope:   scope,
        PlayerName:  parts[0],
        Message: parts[1],
    }, true
}