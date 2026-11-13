package ui

import (
	"github.com/charmbracelet/lipgloss"
	"the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) View() string {
    if m.appState == StateAuthentication {
        return m.renderAuthentication()
    }

    return m.renderView()
}

func (m Model) renderAuthentication() string {
    title := layout.TitleStyle.Render("THE ANSWER PROTOCOL")

    content := lipgloss.JoinVertical(
        lipgloss.Left,
        title,
        "",
        "Welcome, adventurer!",
        "",
        "Enter your username:",
        "",
        layout.InputStyle.Render("> "+m.input),
        "",
        layout.StatusStyle.Render(m.status),
    )

    return layout.PanelStyle.Render(content)
}

func (m Model) renderView() string {
	var content string

	switch m.currentView {
	case ViewWorld:
		content = m.renderWorld()
	case ViewRoom:
		content = renderRoom(m.currentRoom)
	case ViewInventory:
		content = m.renderInventory()
	case ViewCharacter:
		content = m.renderCharacter()
	case ViewQuests:
		content = m.renderQuests()
	case ViewGroup:
		content = m.renderGroup()
	case ViewCombat:
		content = m.renderCombat()
	default:
		content = "Unknown view"
	}

	header := layout.RenderHeader(
		m.username,
		m.playerStatus.HP,
		m.playerStatus.MaxHP,
	)

	sidebar := layout.RenderSidebar(
		m.renderQuests(),
		m.renderGroup(),
	)

	eventLog := m.renderEventLog()
	input := layout.InputStyle.Render("> " + m.input)
	status := layout.StatusStyle.Render(m.status)

	if m.width == 0 || m.height == 0 {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			content,
			eventLog,
			input,
			status,
		)
	}

	availableHeight := m.height -
		lipgloss.Height(header) -
		lipgloss.Height(eventLog) -
		lipgloss.Height(input) -
		lipgloss.Height(status)

	body := layout.RenderMain(
		content,
		sidebar,
		m.width,
		availableHeight,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		body,
		eventLog,
		input,
		status,
	)
}

func (m Model) renderWorld() string {
    return renderRoom(m.currentRoom)
}

func (m Model) renderInventory() string {
	return "Inventory View"
}

func (m Model) renderCharacter() string {
	return "Character View"
}

func (m Model) renderQuests() string {
	return "Quests View"
}

func (m Model) renderGroup() string {
	return "Group View"
}

func (m Model) renderCombat() string {
	return "Combat View"
}

func (m Model) renderEventLog() string {
    title := layout.TitleStyle.Render("EVENT LOG")
    content := title + "\n\n"

    visibleEvents := layout.EventLogHeight - 4

    start := 0
    if len(m.eventLog) > visibleEvents {
        start = len(m.eventLog) - visibleEvents
    }

    for _, event := range m.eventLog[start:] {
        content += event + "\n"
    }

    return layout.PanelStyle.
        Height(layout.EventLogHeight - 4).
        Render(content)
}