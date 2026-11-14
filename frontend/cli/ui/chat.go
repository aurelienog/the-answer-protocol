
package ui

import (
    "strings"

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
        content.WriteString(
            "[" + message.Scope + "] " +
                message.PlayerName + ": " +
                message.Message + "\n",
        )
    }

    contentWidth := width - layout.PanelWidthOffset
    if contentWidth < 0 {
        contentWidth = 0
    }

    panelHeight := height - layout.PanelHeightOffset
    if panelHeight < 0 {
        panelHeight = 0
    }

    return layout.PanelStyle.
        Width(contentWidth).
        Height(panelHeight).
        Render(content.String())
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
