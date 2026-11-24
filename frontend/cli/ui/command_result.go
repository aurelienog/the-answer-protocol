package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"the-answer-protocol/internal/protocol"
)

type commandHandler func(*Model, string) error
type errorHandler func(*Model)

var commandHandlers = map[string]commandHandler{
	"STATUS":            handleStatusResponse,
	"WHO":               handleWhoResponse,
	"QUESTS":            handleQuestsResponse,
	"QUEST":             handleQuestResponse,
	"QUIT":              handleActionResponse,
	"USE":               handleUseResponse,
	"GROUP INFO":        handleGroupInfoResponse,
	"GROUP CREATE":      handleActionResponse,
	"GROUP JOIN":        handleActionResponse,
	"GROUP INVITE":      handleActionResponse,
	"GROUP INVITATIONS": handleGroupInvitationsResponse,
	"GROUP LEAVE":       handleActionResponse,
	"LOOK":              handleLookResponse,
	"MOVE":              handleMoveResponse,
	"TALK":              handleTalkResponse,
	"INVENTORY":         handleInventoryResponse,
	"TAKE":              handleTakeResponse,
	"DROP":              handleDropResponse,
	"ATTACK":            handleAttackResponse,
	"COMBAT ATTACK":     handleCombatActionResponse,
	"COMBAT DEFEND":     handleCombatActionResponse,
	"COMBAT USE":        handleCombatUseResponse,
	"COMBAT FLEE":       handleCombatFleeResponse,
	"COMBAT HELP":       handleCombatActionResponse,
	"COMBAT JOIN":       handleCombatActionResponse,
}

var commandErrorHandlers = map[string]errorHandler{
	"NOT_IN_GROUP": func(m *Model) {
		m.world.Group = Group{}
	},
}

func handleActionResponse(_ *Model, _ string) error {
	return nil
}

func (m Model) handleCommandResult(msg commandResultMsg) (tea.Model, tea.Cmd) {
	// Handle transport and client errors first.
	if msg.err != nil {
		m.ui.PendingRefresh = RefreshNone
		m.communication.StatusMessage = "Error: " + msg.err.Error()
		return m, nil
	}

	// Handle server error responses.
	if msg.response.Kind == protocol.KindERR {
		m.ui.PendingRefresh = RefreshNone

		errorCode := strings.ToUpper(strings.TrimSpace(msg.response.Name))
		m.communication.StatusMessage = errorCode

		if handler, exists := commandErrorHandlers[errorCode]; exists {
			handler(&m)
		}

		return m, nil
	}

	command := getCommandName(msg.command)

	if msg.response.Kind != protocol.KindOK {
		m.ui.PendingRefresh = RefreshNone
		m.communication.StatusMessage = "Unexpected server response"
		return m, nil
	}

	handler, exists := commandHandlers[command]
	if !exists {
		m.communication.StatusMessage = "Command executed"
		return m, nil
	}

	pendingRefresh := m.ui.PendingRefresh

	if err := handler(&m, msg.response.Payload); err != nil {
		m.ui.PendingRefresh = RefreshNone
		m.communication.StatusMessage = "Error parsing " + command + ": " + err.Error()
		return m, nil
	}

	if command == "GROUP JOIN" {
		parts := strings.Fields(msg.command)
		if len(parts) > 2 {
			m.player.Invitations = removeInvitationByName(
				m.player.Invitations,
				strings.Join(parts[2:], " "),
			)
		}
	}

	// Handle commands that navigate to another view.
	switch command {
	case "LOOK":
		if pendingRefresh != RefreshRoom {
			return m, m.startViewTransition(ViewRoom)
		}

	case "INVENTORY":
		if pendingRefresh != RefreshInventory {
			return m, m.startViewTransition(ViewInventory)
		}

	case "QUESTS":
		if pendingRefresh != RefreshQuests {
			return m, m.startViewTransition(ViewQuests)
		}

	case "STATUS":
		if pendingRefresh != RefreshNone {
			return m, nil
		}
		return m, m.startViewTransition(ViewCharacter)

	case "TALK":
		return m, m.startViewTransition(ViewTalk)

	case "GROUP INFO", "GROUP INVITATIONS":
		if pendingRefresh != RefreshGroup {
			return m, m.startViewTransition(ViewGroup)
		}
	}

	if command == "QUEST" {
		m.communication.StatusMessage = "Quest accepted. Updating quests..."
		m.ui.PendingRefresh = RefreshQuests
		return m, sendCommand(m.client, "QUESTS")
	}

	if command == "DROP" || command == "TAKE" {
		m.communication.StatusMessage = "Item updated. Refreshing data..."

		switch m.navigation.CurrentView {
		case ViewInventory:
			m.ui.PendingRefresh = RefreshInventory
			return m, sendCommand(m.client, "INVENTORY")

		case ViewRoom, ViewWorld:
			m.ui.PendingRefresh = RefreshRoom
			return m, sendCommand(m.client, "LOOK")
		}
	}

	// Refresh group information after group operations.
	switch command {
	case "GROUP CREATE", "GROUP JOIN":
		m.communication.StatusMessage = "Group updated. Loading group..."
		m.ui.PendingRefresh = RefreshGroup
		return m, sendCommand(m.client, "GROUP INFO")

	case "GROUP INVITE":
		m.communication.StatusMessage = "Invitation sent."
		m.ui.PendingRefresh = RefreshGroup
		return m, sendCommand(m.client, "GROUP INFO")

	case "GROUP INVITATIONS":
		m.communication.StatusMessage = "Invitations updated."
		return m, nil

	case "GROUP LEAVE":
		m.world.Group = Group{}
		m.communication.StatusMessage = "You left the group."
		return m, nil
	}

	if command == "MOVE" {
		m.communication.StatusMessage = "Movement successful. Updating room..."
		return m, sendCommand(m.client, "LOOK")
	}

	if command == "QUIT" {
		return m, tea.Quit
	}

	if command == "WHO" {
		m.communication.StatusMessage = fmt.Sprintf(
			"Players online: %d",
			m.world.OnlinePlayers,
		)
		return m, nil
	}

	// Combat responses: UI messages, view transition, and background refreshes.
	switch command {
	case "ATTACK", "COMBAT ATTACK", "COMBAT JOIN":
		m.communication.StatusMessage = "Combat active."
		if m.navigation.CurrentView != ViewCombat {
			return m, m.startViewTransition(ViewCombat)
		}
		return m, nil

	case "COMBAT DEFEND":
		fields := parsePayloadFields(msg.response.Payload)
		m.communication.StatusMessage = fmt.Sprintf(
			"Defending (+%s%% defense)",
			fields["defense_bonus"],
		)
		return m, nil

	case "COMBAT FLEE":
		m.communication.StatusMessage = "You fled from combat!"
		return m, nil

	case "USE":
		fields := parsePayloadFields(msg.response.Payload)
		m.communication.StatusMessage = "Used " + fields["used"]
		m.ui.PendingRefresh = RefreshInventory
		return m, sendCommand(m.client, "INVENTORY")

	case "COMBAT USE":
		fields := parsePayloadFields(msg.response.Payload)
		m.communication.StatusMessage = "Used " + fields["used"] + " during combat"
		m.ui.PendingRefresh = RefreshInventory
		return m, sendCommand(m.client, "INVENTORY")
	}

	m.communication.StatusMessage = command + " updated"
	return m, nil
}

func getCommandName(rawCommand string) string {
	parts := strings.Fields(strings.ToUpper(rawCommand))

	if len(parts) == 0 {
		return ""
	}

	if len(parts) >= 2 {
		twoWords := parts[0] + " " + parts[1]
		if _, exists := commandHandlers[twoWords]; exists {
			return twoWords
		}
	}

	return parts[0]
}
