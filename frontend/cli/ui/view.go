package ui

import (
	"github.com/charmbracelet/lipgloss"
	"the-answer-protocol/frontend/cli/ui/layout"
)

var (
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	inputStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42"))
)

func (m Model) View() string {
    if m.appState == StateAuthentication {
        return m.renderAuthentication()
    }

    return m.renderView()
}

func (m Model) renderAuthentication() string {
    title := panelStyle.Render(
        "THE ANSWER PROTOCOL\n\n" +
            "Welcome, adventurer!\n\n" +
            "Enter your username:\n\n" +
            "> " + m.input + "\n\n" +
            m.status,
    )

    return title
}

func (m Model) renderView() string {
	var content string

	switch m.currentView {
	case ViewWorld:
		content = m.renderWorld()
	case ViewRoom:
		content = m.renderRoom()
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
		m.renderCharacter(),
		m.renderQuests(),
	)

	eventLog := m.renderEventLog()
	input := inputStyle.Render("> " + m.input)
	status := statusStyle.Render(m.status)

	if m.width == 0 {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			content,
			eventLog,
			input,
			status,
		)
	}

	body := layout.RenderMain(content, sidebar, m.width)

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
	return "World View"
}

func (m Model) renderRoom() string {
	return "Room View"
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
    content := "EVENT LOG\n\n"

    start := 0
    if len(m.eventLog) > 8 {
        start = len(m.eventLog) - 8
    }

    for _, event := range m.eventLog[start:] {
        content += event + "\n"
    }

    return panelStyle.Render(content)
}