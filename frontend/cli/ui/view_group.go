
package ui

import (
	"sort"

	"github.com/charmbracelet/lipgloss"

	"the-answer-protocol/frontend/cli/ui/layout"
)

func (m Model) renderGroup(detailed bool) string {
	var sections []string

	group := m.world.Group

	// Group header
	if group.ID == "" {
		sections = append(
			sections,
			layout.TitleStyle.Render(layout.IconGroup+" GROUP"),
			"",
			layout.MutedStyle.Render("You are not in a group"),
		)
	} else {
		sections = append(
			sections,
			layout.TitleStyle.Render(
				layout.IconGroup+" GROUP "+group.Name,
			),
		)

		if detailed {
			sections = append(
				sections,
				layout.MutedStyle.Render("ID: "+friendlyID(group.ID)),
				"",
				layout.TitleStyle.Render("MEMBERS"),
			)
		} else {
			sections = append(sections, "")
		}

		// Group members
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

		// Invitations sent by the current group
		if detailed {
			sections = append(
				sections,
				"",
				layout.TitleStyle.Render("PENDING INVITATIONS"),
			)

			pendingInvitations := make([]string, 0)

			for player, pending := range group.Invitations {
				if pending {
					pendingInvitations = append(
						pendingInvitations,
						player,
					)
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
	}

	// Invitations received by the current player.
	// This section is independent of group membership.
	if detailed {
		sections = append(
			sections,
			"",
			layout.TitleStyle.Render("RECEIVED INVITATIONS"),
		)

		if len(m.player.Invitations) == 0 {
			sections = append(
				sections,
				layout.MutedStyle.Render("No received invitations"),
			)
		} else {
			invitations := append(
				[]GroupInvitation(nil),
				m.player.Invitations...,
			)

			sort.Slice(invitations, func(i, j int) bool {
				return invitations[i].Name < invitations[j].Name
			})

			for _, invitation := range invitations {
				sections = append(
					sections,
					layout.WarningStyle.Render(
						"[+] "+invitation.Name,
					),
				)
			}
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		sections...,
	)
}
