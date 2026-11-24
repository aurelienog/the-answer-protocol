package ui

import (
    "strings"

    "github.com/charmbracelet/lipgloss"
    "the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) renderChat(width, height int) string {
    messages := m.filteredChatMessages()
    entries := make([]string, 0, len(messages))
    for _, message := range messages {
        entries = append(entries, renderChatMessage(message))
    }

    title := layout.TitleStyle.Render("CHAT") + "  " + m.renderChatFilter()
    return renderRecentPanel(title, entries, width, height, 1)
}

func (m Model) renderChatFilter() string {
    filters := []struct {
        filter ChatFilter
        label  string
    }{
        {ChatFilterAll, "ALL"},
        {ChatFilterGlobal, "GLOBAL"},
        {ChatFilterRoom, "ROOM"},
        {ChatFilterGroup, "GROUP"},
    }

    var options []string

    for _, filter := range filters {
        label := filter.label

        if m.communication.ChatFilter == filter.filter {
            label = layout.TitleStyle.Render("[" + label + "]")
        } else {
            label = "[" + label + "]"
        }

        options = append(options, label)
    }

    return strings.Join(options, " ")
}

func (m Model) filteredChatMessages() []ChatMessage {
    messages := make([]ChatMessage, 0, len(m.communication.ChatLog))

    for _, message := range m.communication.ChatLog {
        if m.matchesChatFilter(message) {
            messages = append(messages, message)
        }
    }

    return messages
}

func (m Model) matchesChatFilter(message ChatMessage) bool {
    switch m.communication.ChatFilter {
    case ChatFilterAll:
        return true
    case ChatFilterGlobal:
        return message.Scope == "GLOBAL"
    case ChatFilterRoom:
        return message.Scope == "ROOM"
    case ChatFilterGroup:
        return message.Scope == "GROUP"
    default:
        return true
    }
}

func renderChatMessage(message ChatMessage) string {
    var scopeStyle lipgloss.Style

    switch strings.ToUpper(message.Scope) {
    case "ROOM":
        scopeStyle = layout.RoomChatStyle
    case "GLOBAL":
        scopeStyle = layout.GlobalChatStyle
    case "GROUP":
        scopeStyle = layout.GroupChatStyle
    default:
        scopeStyle = layout.MutedStyle
    }

    tag := scopeStyle.Render("[" + strings.ToUpper(message.Scope) + "]")
    text := layout.MutedStyle.Render(message.Message)

    return tag + " " + message.PlayerName + ": " + text
}