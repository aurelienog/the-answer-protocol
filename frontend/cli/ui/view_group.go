
package ui

import (
	"github.com/charmbracelet/lipgloss"

	"the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) renderGroup() string {
	group := m.world.Group

	if group.ID == "" {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			layout.TitleStyle.Render(layout.IconGroup+" GROUP"),
			"",
			layout.StatusStyle.Render("You are not in a group"),
		)
	}

	sections := []string{
		layout.TitleStyle.Render(layout.IconGroup+" GROUP"),
		layout.TitleStyle.Render(group.Name),
	}

	if group.ID != "" {
		sections = append(
			sections,
			layout.StatusStyle.Render("ID: "+group.ID),
		)
	}

	sections = append(sections, "")

	sections = append(
		sections,
		layout.TitleStyle.Render("MEMBERS"),
	)

	if len(group.Members) == 0 {
		sections = append(
			sections,
			layout.StatusStyle.Render("No members"),
		)
	} else {
		for _, member := range group.Members {
			sections = append(
				sections,
				layout.SuccessStyle.Render("[+] "+member),
			)
		}
	}

	pendingInvitations := make([]string, 0)

	for player, pending := range group.Invitations {
		if pending {
			pendingInvitations = append(pendingInvitations, player)
		}
	}

	sections = append(sections, "")
	sections = append(
		sections,
		layout.TitleStyle.Render("PENDING INVITATIONS"),
	)

	if len(pendingInvitations) == 0 {
		sections = append(
			sections,
			layout.StatusStyle.Render("No pending invitations"),
		)
	} else {
		for _, player := range pendingInvitations {
			sections = append(
				sections,
				layout.WarningStyle.Render("[?] "+player),
			)
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		sections...,
	)
}
