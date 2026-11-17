package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) View() string {
	if m.application.Current == StateAuthentication {
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
		layout.InputStyle.Render(m.inputText()),
		"",
		layout.StatusStyle.Render(m.communication.StatusMessage),
	)

	return layout.PanelStyle.Render(content)
}

func (m Model) renderCurrentView() string {
    switch m.navigation.CurrentView {
	case ViewWorld:
        return m.renderWorld()
    case ViewRoom:
        return renderRoom(
			m.world.Room,
			m.ui.MainViewport.Width,
			m.ui.MainViewport.Height,
		)
    case ViewInventory:
        return m.renderInventory()
    case ViewCharacter:
        return m.renderCharacter()
	case ViewTalk:
    	return m.renderTalk()
    case ViewQuests:
        return m.renderQuests(true, m.ui.MainViewport.Width)
    case ViewGroup:
        return m.renderGroup(true)
    case ViewCombat:
        return m.renderCombat()
    default:
        return m.renderWorld()
    }
}

func (m Model) renderView() string {
	result := m.calculateLayout()

	if !result.MainView.Visible {
		return layout.WarningStyle.Render("Terminal too small")
	}

	// All content panels share the same outer width.
	contentWidth := result.MainView.Width + result.Sidebar.Width

	header := layout.RenderHeader(
		m.player.Username,
		m.player.Status.HP,
		m.player.Status.MaxHP,
		contentWidth,
	)

	sidebarContent := ""
	if result.Sidebar.Visible {
		sidebarContent = m.ui.SidebarViewport.View()
	}

	mainContent := m.ui.MainViewport.View()

	if m.ui.Transitioning &&
		m.ui.TransitionPhase == TransitionReveal {

		mainContent = lipgloss.Place(
			m.ui.MainViewport.Width,
			m.ui.MainViewport.Height,
			lipgloss.Center,
			lipgloss.Center,
			layout.RenderSeparator(m.ui.MainViewport.Width),
		)
	}

	body := layout.RenderMain(
		mainContent,
		sidebarContent,
		result.MainView.Width,
		result.Sidebar.Width,
		result.MainView.Height,
		result.Sidebar.Visible,
		m.ui.PanelFocused,
	)

	sections := []string{
		header,
		body,
	}

	// Event Log and Chat share the same available width as Main View and Sidebar.
	if result.EventLog.Visible || result.Chat.Visible {
		var bottomPanels []string

		if result.EventLog.Visible {
			bottomPanels = append(
				bottomPanels,
				m.renderEventLog(
					result.EventLog.Width,
					result.EventLog.Height,
				),
			)
		}

		if result.Chat.Visible {
			bottomPanels = append(
				bottomPanels,
				m.renderChat(
					result.Chat.Width,
					result.Chat.Height,
				),
			)
		}

		sections = append(
			sections,
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				bottomPanels...,
			),
		)
	}

	// Available Commands uses the same outer width.
	if result.AvailableCommands.Visible {
		sections = append(
			sections,
			m.renderAvailableCommands(contentWidth),
		)
	}

	// Input and Status remain outside the content panels.
	input := layout.InputStyle.
		Width(contentWidth).
		Render(m.inputText())

	status := layout.StatusStyle.
		Width(contentWidth).
		Render(m.communication.StatusMessage)

	sections = append(sections, input, status)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		sections...,
	)
}

func (m Model) renderCombat() string {
	return "Combat View"
}

func (m Model) renderEventLog(width, height int) string {
	return renderRecentPanel(
		layout.TitleStyle.Render("EVENT LOG"),
		m.communication.EventLog,
		width,
		height,
		0,
	)
}

func renderRecentPanel(
	title string,
	entries []string,
	width, height, gap int,
) string {
	contentWidth := maxInt(0, width-layout.PanelWidthOffset)
	contentLines := maxInt(0, height-layout.PanelHeightOffset)

	renderedTitle := lipgloss.NewStyle().
		Width(contentWidth).
		Render(title)
	titleLines := strings.Split(renderedTitle, "\n")
	remainingLines := maxInt(0, contentLines-len(titleLines)-gap)

	var recentEntries [][]string

	for i := len(entries) - 1; i >= 0 && remainingLines > 0; i-- {
		renderedEntry := lipgloss.NewStyle().
			Width(contentWidth).
			Render(entries[i])
		entryLines := strings.Split(renderedEntry, "\n")

		if len(entryLines) > remainingLines {
			entryLines = entryLines[len(entryLines)-remainingLines:]
		}

		recentEntries = append(recentEntries, entryLines)
		remainingLines -= len(entryLines)
	}

	var content strings.Builder
	content.WriteString(renderedTitle)
	content.WriteString(strings.Repeat("\n", gap+1))

	for i := len(recentEntries) - 1; i >= 0; i-- {
		if i != len(recentEntries)-1 {
			content.WriteString("\n")
		}
		content.WriteString(strings.Join(recentEntries[i], "\n"))
	}

	return layout.RenderPanel(content.String(), width, height)
}

func (m Model) renderAvailableCommands(width int) string {
	panelWidth := maxInt(0, width-layout.PanelBorderWidth)

	return layout.PanelStyle.
		Width(panelWidth).
		Render(m.availableCommandsContent())
}

func (m Model) inputText() string {
	input := "> " + m.ui.Input

	if m.ui.CursorVisible {
		input += "█"
	}

	return input
}