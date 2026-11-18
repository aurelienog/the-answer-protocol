
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) renderCombat() string {
	width := m.ui.MainViewport.Width

	if width <= 0 {
		return ""
	}

	combat := m.world.Combat

	title := layout.TitleStyle.Render("⚔ COMBAT")

	sections := []string{
		title,
		"",
		renderCombatInfo(combat),
		"",
		renderCombatParticipants(combat.Participants, width),
		"",
		renderCombatTurn(combat),
		"",
		renderCombatLog(combat.Log, width),
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func renderCombatInfo(combat CombatState) string {
	if combat.ID == "" {
		return layout.StatusStyle.Render("No active combat")
	}

	status := layout.SuccessStyle.Render("ACTIVE")
	if !combat.Active {
		status = layout.MutedStyle.Render("ENDED")
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		fmt.Sprintf("%s %s", layout.StatusStyle.Render("Combat:"), layout.InfoStyle.Render(combat.ID)),
		fmt.Sprintf("%s %d", layout.StatusStyle.Render("Round:"), combat.Round),
		fmt.Sprintf("%s %s", layout.StatusStyle.Render("Status:"), status),
	)
}

func renderCombatParticipants(participants []CombatParticipantState, width int) string {
	lines := []string{
		layout.TitleStyle.Render("PARTICIPANTS"),
		layout.RenderSeparator(width),
	}

	if len(participants) == 0 {
		lines = append(lines, layout.StatusStyle.Render("No participants"))
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	for _, participant := range participants {
		lines = append(lines, renderCombatParticipant(participant, width))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func renderCombatParticipant(participant CombatParticipantState, width int) string {
	name := participant.Name
	if name == "" {
		name = participant.ID
	}

	participantType := "NPC"
	nameStyle := layout.DangerStyle

	if participant.IsPlayer {
		participantType = "PLAYER"
		nameStyle = layout.InfoStyle
	}

	state := "Alive"
	stateStyle := layout.SuccessStyle

	switch {
	case participant.Escaped:
		state = "Escaped"
		stateStyle = layout.WarningStyle
	case !participant.Alive:
		state = "Dead"
		stateStyle = layout.DangerStyle
	case !participant.Connected && participant.IsPlayer:
		state = "Disconnected"
		stateStyle = layout.WarningStyle
	}

	header := fmt.Sprintf(
		"%s %s %s",
		nameStyle.Render(name),
		layout.MutedStyle.Render("["+participantType+"]"),
		stateStyle.Render("("+state+")"),
	)

	lines := []string{header}

	if participant.MaxHP > 0 {
		health := fmt.Sprintf(
			"HP: %d/%d",
			participant.HP,
			participant.MaxHP,
		)

		healthStyle := layout.SuccessStyle
		if participant.HP <= participant.MaxHP/4 {
			healthStyle = layout.DangerStyle
		}

		lines = append(lines, healthStyle.Render(health))
	}

	return lipgloss.NewStyle().
		Width(width).
		Render(strings.Join(lines, "\n"))
}

func renderCombatTurn(combat CombatState) string {
	lines := []string{
		layout.TitleStyle.Render("CURRENT TURN"),
	}

	if !combat.Active {
		lines = append(lines, layout.StatusStyle.Render("Combat has ended"))
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	if combat.CurrentTurn == "" {
		lines = append(lines, layout.StatusStyle.Render("Waiting for turn information"))
	} else {
		lines = append(
			lines,
			fmt.Sprintf(
				"%s %s",
				layout.StatusStyle.Render("Participant:"),
				layout.InfoStyle.Render(combat.CurrentTurn),
			),
			fmt.Sprintf(
				"%s %d",
				layout.StatusStyle.Render("Available actions:"),
				combat.AvailableActions,
			),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func renderCombatLog(log []string, width int) string {
	lines := []string{
		layout.TitleStyle.Render("COMBAT LOG"),
		layout.RenderSeparator(width),
	}

	if len(log) == 0 {
		lines = append(lines, layout.StatusStyle.Render("No combat events yet"))
	} else {
		for _, entry := range log {
			lines = append(
				lines,
				layout.WrapText(entry, width),
			)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
