
package ui

import (
    "strings"

    "github.com/charmbracelet/lipgloss"
    "the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) renderChat(width, height int) string {
    title := layout.TitleStyle.Render("CHAT")
    filters := m.renderChatFilter()

    var content strings.Builder
    content.WriteString(title)
    content.WriteString("  ")
    content.WriteString(filters)
    content.WriteString("\n\n")

    var messages []ChatMessage

    for _, message := range m.communication.ChatLog {
        if m.matchesChatFilter(message) {
            messages = append(messages, message)
        }
    }

    visibleEntries := height - layout.PanelHeightOffset - 2

    if visibleEntries < 0 {
        visibleEntries = 0
    }

    start := 0
    if len(messages) > visibleEntries {
        start = len(messages) - visibleEntries
    }

    for _, message := range messages[start:] {
        content.WriteString(renderChatMessage(message))
        content.WriteString("\n")
    }

    return layout.RenderPanel(content.String(), width, height)
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