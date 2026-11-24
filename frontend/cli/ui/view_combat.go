package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"the-answer-protocol/frontend/cli/ui/animations"
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
		m.renderCombatParticipants(combat.Participants, width),
		"",
		m.renderCombatTurn(),
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

	playerStatus := layout.MutedStyle.Render("NOT IN COMBAT")
	if combat.Active {
		playerStatus = layout.SuccessStyle.Render("IN COMBAT")
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		fmt.Sprintf("%s %s", layout.StatusStyle.Render("Combat:"), layout.InfoStyle.Render(combat.ID)),
		fmt.Sprintf("%s %d", layout.StatusStyle.Render("Round:"), combat.Round),
		fmt.Sprintf("%s %s", layout.StatusStyle.Render("Status:"), status),
		fmt.Sprintf("%s %s", layout.StatusStyle.Render("You:"), playerStatus),
	)
}

func (m Model) renderCombatParticipants(participants []CombatParticipantState, width int) string {
	lines := []string{
		layout.TitleStyle.Render("PARTICIPANTS"),
		layout.RenderSeparator(width),
	}

	if len(participants) == 0 {
		lines = append(lines, layout.StatusStyle.Render("No participants"))
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	for _, participant := range participants {
		lines = append(lines, m.renderCombatParticipant(participant, width))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderCombatParticipant(participant CombatParticipantState, width int) string {
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

	turnMarker := "  "
	if m.world.Combat.Active &&
		(strings.EqualFold(m.world.Combat.CurrentTurn, participant.Name) ||
			strings.EqualFold(m.world.Combat.CurrentTurn, participant.ID)) {
		turnMarker = layout.WarningStyle.Render("▶ ")
	}

	header := fmt.Sprintf(
		"%s%s  %s  %s",
		turnMarker,
		nameStyle.Render(name),
		layout.MutedStyle.Render(participantType),
		stateStyle.Render(state),
	)

	lines := []string{header}

	if participant.MaxHP > 0 {
		id := participant.ID
		if id == "" {
			id = participant.Name
		}
		if participant.IsPlayer &&
			(strings.EqualFold(participant.ID, m.player.Username) ||
				strings.EqualFold(participant.Name, m.player.Username)) {
			id = m.player.Username
		}

		bar, exists := m.ui.healthBars[id]

		var health string

		if exists && bar.Initialized {
			health = layout.RenderAnimatedHealthBar(
				bar.Current,
				participant.MaxHP,
				14,
			)
		} else {
			health = layout.RenderHealthBar(
				participant.HP,
				participant.MaxHP,
				14,
			)
		}

		lines = append(lines, "   "+health)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lines...,
	)
}

func (m Model) renderCombatTurn() string {
	combat := m.world.Combat
	if !combat.Active {
		return layout.MutedStyle.Render("You are not in combat")
	}

	if combat.CurrentTurn == "" {
		return layout.StatusStyle.Render("Waiting for turn information")
	}

	if strings.EqualFold(combat.CurrentTurn, m.player.Username) {
		return fmt.Sprintf(
			"%s  %s",
			layout.TitleStyle.Render("CURRENT TURN"),
			layout.SuccessStyle.Render("⚔ YOUR TURN ("+combat.CurrentTurn+")"),
		)
	}

	return fmt.Sprintf(
		"%s  %s",
		layout.TitleStyle.Render("CURRENT TURN"),
		layout.WarningStyle.Render("⏳ "+combat.CurrentTurn),
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

func (m *Model) syncCombatHealthBars() bool {
	if m.ui.healthBars == nil {
		m.ui.healthBars = make(
			map[string]animations.HealthBarAnimation,
		)
	}

	animating := false

	// Sincronizar vida del jugador.
	if m.player.Username != "" && m.player.Status.MaxHP > 0 {
		id := m.player.Username

		bar := m.ui.healthBars[id]

		if bar.SetTarget(
			m.player.Status.HP,
			m.player.Status.MaxHP,
		) {
			animating = true
		}

		m.ui.healthBars[id] = bar
	}

	// Sincronizar participantes del combate.
	for _, participant := range m.world.Combat.Participants {
		id := participant.ID
		if id == "" {
			id = participant.Name
		}
		if participant.IsPlayer &&
			(strings.EqualFold(participant.ID, m.player.Username) ||
				strings.EqualFold(participant.Name, m.player.Username)) {
			id = m.player.Username
		}

		if id == "" || participant.MaxHP <= 0 {
			continue
		}

		bar := m.ui.healthBars[id]

		if bar.SetTarget(participant.HP, participant.MaxHP) {
			animating = true
		}

		m.ui.healthBars[id] = bar
	}

	return animating
}
