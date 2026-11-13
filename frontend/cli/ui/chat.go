
package ui

import (
    "strings"

    "the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) renderChat(width int) string {
    title := layout.TitleStyle.Render("CHAT")

    var content strings.Builder
    content.WriteString(title)
    content.WriteString("\n")

    visibleEntries := m.eventLogHeight -
        layout.PanelHeightOffset -
        1

    if visibleEntries < 0 {
        visibleEntries = 0
    }

    start := 0
    if len(m.chatLog) > visibleEntries {
        start = len(m.chatLog) - visibleEntries
    }

    for _, message := range m.chatLog[start:] {
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

    return layout.PanelStyle.
        Width(contentWidth).
        Height(m.eventLogHeight - layout.PanelHeightOffset).
        Render(content.String())
}
