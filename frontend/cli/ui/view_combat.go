
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
		renderCombatLog(
			combat.Log,
			width,
			combat.CurrentTurn,
		),
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

	state := "ALIVE"
	stateStyle := layout.SuccessStyle

	switch {
	case participant.Escaped:
		state = "ESCAPED"
		stateStyle = layout.WarningStyle
	case !participant.Alive:
		state = "DEAD"
		stateStyle = layout.DangerStyle
	case !participant.Connected && participant.IsPlayer:
		state = "DISCONNECTED"
		stateStyle = layout.WarningStyle
	}

	header := fmt.Sprintf(
		"%s  %s  %s",
		nameStyle.Render(name),
		layout.MutedStyle.Render(participantType),
		stateStyle.Render(state),
	)

	lines := []string{header}

	if participant.MaxHP > 0 {
		health := layout.RenderHealthBar(
			participant.HP,
			participant.MaxHP,
			14,
		)

		lines = append(lines, "   "+health)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lines...,
	)
}

func renderCombatTurn(combat CombatState) string {
	if !combat.Active {
		return layout.MutedStyle.Render("Combat has ended")
	}

	if combat.CurrentTurn == "" {
		return layout.StatusStyle.Render("Waiting for turn information")
	}

	return fmt.Sprintf(
		"%s  %s",
		layout.TitleStyle.Render("CURRENT TURN"),
		layout.InfoStyle.Render("⚔ "+combat.CurrentTurn),
	)
}

func renderCombatLog(
	log []string,
	width int,
	currentTurn string,
) string {
	lines := []string{
		layout.TitleStyle.Render("BATTLE CHRONICLE"),
		layout.RenderSeparator(width),
	}

	if len(log) == 0 {
		lines = append(
			lines,
			layout.StatusStyle.Render("No combat events yet"),
		)
	} else {
		for _, entry := range log {
			styledEntry := styleCombatEvent(entry, currentTurn)

			lines = append(
				lines,
				layout.WrapText(styledEntry, width),
			)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func styleCombatEvent(entry string, currentTurn string) string {
	event := strings.ToLower(strings.TrimSpace(entry))

	switch {
	case strings.Contains(event, "combat started"):
		return layout.CombatStartStyle.Render("✦ " + entry)

	case strings.Contains(event, "combat ended"):
		return layout.CombatVictoryStyle.Render("✦ " + entry)

	case strings.HasPrefix(event, "turn:"):
		turn := strings.TrimSpace(entry[len("Turn:"):])

		if strings.EqualFold(turn, currentTurn) {
			return layout.TitleStyle.Render("◆ " + entry)
		}

		return layout.CombatTurnStyle.Render("◆ " + entry)

	case strings.Contains(event, " attacks "):
		return layout.CombatAttackStyle.Render("⚔ " + entry)

	case strings.Contains(event, " deals ") &&
		strings.Contains(event, " damage "):
		return layout.CombatDamageStyle.Render("▼ " + entry)

	case strings.Contains(event, " heals ") ||
		strings.Contains(event, " restores "):
		return layout.CombatHealingStyle.Render("✚ " + entry)

	case strings.Contains(event, " misses ") ||
		strings.Contains(event, " dodges "):
		return layout.CombatMissStyle.Render("◇ " + entry)

	case strings.Contains(event, " defeated ") ||
		strings.Contains(event, " killed ") ||
		strings.Contains(event, " died"):
		return layout.CombatVictoryStyle.Render("☠ " + entry)

	default:
		return layout.CombatInfoStyle.Render("· " + entry)
	}
}