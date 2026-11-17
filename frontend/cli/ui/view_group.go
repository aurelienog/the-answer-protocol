
package ui

import (
	"sort"

	"github.com/charmbracelet/lipgloss"

	"the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) renderGroup(detailed bool) string {
	group := m.world.Group

	if group.ID == "" {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			layout.TitleStyle.Render(layout.IconGroup+" GROUP"),
			"",
			layout.MutedStyle.Render("You are not in a group"),
		)
	}

	title := layout.TitleStyle.Render(
		layout.IconGroup + " GROUP " + group.Name,
	)

	sections := []string{
		title,
	}

	if detailed {
		sections = append(
			sections,
			layout.MutedStyle.Render("ID: "+group.ID),
			"",
			layout.TitleStyle.Render("MEMBERS"),
		)
	} else {
		sections = append(
			sections,
			"",
		)
	}

	if len(group.Members) == 0 {
		sections = append(
			sections,
			layout.MutedStyle.Render("No members"),
		)
	} else {
		for _, member := range group.Members {
			sections = append(
				sections,
				layout.SuccessStyle.Render("[+] "+member),
			)
		}
	}

	if detailed {
		sections = append(
			sections,
			"",
			layout.TitleStyle.Render("PENDING INVITATIONS"),
		)

		pendingInvitations := make([]string, 0)

		for player, pending := range group.Invitations {
			if pending {
				pendingInvitations = append(pendingInvitations, player)
			}
		}

		sort.Strings(pendingInvitations)

		if len(pendingInvitations) == 0 {
			sections = append(
				sections,
				layout.MutedStyle.Render("No pending invitations"),
			)
		} else {
			for _, player := range pendingInvitations {
				sections = append(
					sections,
					layout.WarningStyle.Render("[?] "+player),
				)
			}
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		sections...,
	)
}
