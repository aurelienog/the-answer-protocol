package ui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"the-answer-protocol/internal/protocol"
)

const maxEventLogEntries = 100

func (m Model) handleServerEvent(
	msg serverEventMsg,
) (tea.Model, tea.Cmd) {
	event := msg.event.Message

	var cmd tea.Cmd

	if event.Kind == protocol.KindEVT && isChatEvent(event.Data) {
		chatMessage, ok := parseChatMessage(event.EventType, event.Data)
		if ok {
			m.communication.ChatLog = append(
				m.communication.ChatLog,
				chatMessage,
			)

			if len(m.communication.ChatLog) > maxEventLogEntries {
				m.communication.ChatLog = m.communication.ChatLog[len(m.communication.ChatLog)-maxEventLogEntries:]
			}
		}
	} else {
		m.appendEventLog(msg.event.Raw)

		if event.Kind == protocol.KindEVT {
			switch event.EventType {
			case "COMBAT":
				combatCmd := m.handleCombatEvent(event.Data)
				if m.world.Combat.Active && m.navigation.CurrentView != ViewCombat {
					cmd = tea.Batch(combatCmd, m.startViewTransition(ViewCombat))
				} else {
					cmd = combatCmd
				}

			case "RESPAWN":
				cmd = m.handleRespawnEvent(event.Data)

			case "GROUP":
				cmd = m.handleGroupEvent(event.Data)
			}
		}
	}

	return m, tea.Batch(
		cmd,
		waitForServerEvent(m.client),
	)
}

func isChatEvent(data string) bool {
	return strings.HasPrefix(data, "CHAT player=")
}

func formatChatMessage(scope, data string) string {
	data = strings.TrimPrefix(data, "CHAT player=")

	parts := strings.SplitN(data, " message=", 2)
	if len(parts) != 2 {
		return data
	}

	return "[" + scope + "] " + parts[0] + ": " + parts[1]
}

func parseChatMessage(scope, data string) (ChatMessage, bool) {
	data = strings.TrimPrefix(data, "CHAT player=")

	parts := strings.SplitN(data, " message=", 2)
	if len(parts) != 2 {
		return ChatMessage{}, false
	}

	return ChatMessage{
		Scope:      scope,
		PlayerName: parts[0],
		Message:    parts[1],
	}, true
}

func (m *Model) appendEventLog(message string) {
	m.communication.EventLog = append(
		m.communication.EventLog,
		message,
	)

	if len(m.communication.EventLog) > maxEventLogEntries {
		m.communication.EventLog = m.communication.EventLog[len(m.communication.EventLog)-maxEventLogEntries:]
	}
}


func (m *Model) handleGroupEvent(data string) tea.Cmd {
	parts := strings.Fields(data)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "INVITE":
		fields := parseKeyValuesOrdered(data, "player", "group")

		player := fields["player"]
		groupName := fields["group"]

		// Ignore invitations not addressed to the current player.
		if player == "" || groupName == "" {
			return nil
		}

		// Avoid adding duplicate invitations.
		for _, invitation := range m.player.Invitations {
			if strings.EqualFold(invitation.Name, groupName) {
				return nil
			}
		}

		m.player.Invitations = append(
			m.player.Invitations,
			GroupInvitation{
				Name: groupName,
			},
		)

		m.communication.StatusMessage =
			"You received an invitation to join " + groupName

	case "JOIN", "LEAVE":
		if m.navigation.CurrentView == ViewGroup {
			m.ui.PendingRefresh = RefreshGroup
			return sendCommand(m.client, "GROUP INFO")
		}
	}

	return nil
}

func (m *Model) appendCombatLog(message string) {
	m.world.Combat.Log = append(
		m.world.Combat.Log,
		message,
	)

	if len(m.world.Combat.Log) > maxEventLogEntries {
		m.world.Combat.Log = m.world.Combat.Log[len(m.world.Combat.Log)-maxEventLogEntries:]
	}
}

// combatEventParticipant mirrors world.CombatParticipantInfo from EVT COMBAT STARTED and COMBAT JOIN.
type combatEventParticipant struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsPlayer  bool   `json:"is_player"`
	HP        *int   `json:"hp"`
	CurrentHP *int   `json:"current_hp"`
	MaxHP     int    `json:"max_hp"`
	Alive     bool   `json:"alive"`
}

func (p combatEventParticipant) resolvedHP() int {
	if p.HP != nil {
		return *p.HP
	}
	if p.CurrentHP != nil {
		return *p.CurrentHP
	}
	return 0
}

func parseKeyValuesOrdered(text string, keys ...string) map[string]string {
	result := make(map[string]string, len(keys))
	for i, key := range keys {
		prefix := key + "="
		start := strings.Index(text, prefix)
		if start == -1 {
			continue
		}
		start += len(prefix)
		end := len(text)
		for _, nextKey := range keys[i+1:] {
			if idx := strings.Index(text[start:], " "+nextKey+"="); idx != -1 {
				end = start + idx
				break
			}
		}
		result[key] = strings.TrimSpace(text[start:end])
	}
	return result
}

func (m *Model) handleCombatEvent(payload string) tea.Cmd {
	parts := strings.SplitN(strings.TrimSpace(payload), " ", 2)
	if len(parts) == 0 {
		return nil
	}

	subType := strings.ToUpper(parts[0])
	body := ""
	if len(parts) > 1 {
		body = parts[1]
	}

	switch subType {
	case "STARTED":
		header, participantsJSON, ok := strings.Cut(body, " participants=")
		if !ok {
			return nil
		}

		headerFields := parsePayloadFields(header)
		combatID := headerFields["combat"]
		if combatID == "" {
			return nil
		}

		var list []combatEventParticipant
		if err := json.Unmarshal([]byte(strings.TrimSpace(participantsJSON)), &list); err != nil {
			m.appendEventLog("Unable to read combat participants.")
			return nil
		}

		isParticipant := false
		participants := make([]CombatParticipantState, 0, len(list))

		for _, p := range list {
			hp := p.resolvedHP()
			participants = append(participants, CombatParticipantState{
				ID:        p.ID,
				Name:      p.Name,
				IsPlayer:  p.IsPlayer,
				Connected: true,
				HP:        hp,
				MaxHP:     p.MaxHP,
				Alive:     p.Alive,
			})

			if p.IsPlayer && (strings.EqualFold(p.ID, m.player.Username) || strings.EqualFold(p.Name, m.player.Username)) {
				isParticipant = true
				m.player.Status.HP = hp
				if p.MaxHP > 0 {
					m.player.Status.MaxHP = p.MaxHP
				}
			}
		}

		if !isParticipant {
			return nil
		}

		m.world.Combat = CombatState{
			ID:           combatID,
			Round:        1,
			Active:       true,
			Participants: participants,
			Log:          []string{"Combat started (" + combatID + ")"},
		}

	case "TURN":
		if !m.world.Combat.Active {
			return nil
		}

		fields := parseKeyValuesOrdered(body, "participant_type", "participant", "round", "actions")
		if round, err := strconv.Atoi(fields["round"]); err == nil && round > 0 {
			m.world.Combat.Round = round
		}
		if actions, err := strconv.Atoi(fields["actions"]); err == nil {
			m.world.Combat.AvailableActions = actions
		}

		participant := fields["participant"]
		m.world.Combat.CurrentTurn = participant
		m.appendCombatLog(fmt.Sprintf("Turn: %s (actions: %d)", participant, m.world.Combat.AvailableActions))

	case "ATTACK":
		if !m.world.Combat.Active {
			return nil
		}

		fields := parseKeyValuesOrdered(body, "attacker_type", "attacker", "target")
		m.appendCombatLog(fmt.Sprintf("%s attacks %s.", fields["attacker"], fields["target"]))

	case "DAMAGE":
		if !m.world.Combat.Active {
			return nil
		}

		fields := parseKeyValuesOrdered(body, "attacker", "target", "amount", "remaining_hp", "hp")
		attacker := fields["attacker"]
		target := fields["target"]
		amount, _ := strconv.Atoi(fields["amount"])

		hpStr := fields["remaining_hp"]
		if hpStr == "" {
			hpStr = fields["hp"]
		}

		if remainingHP, err := strconv.Atoi(hpStr); err == nil {
			m.updateCombatParticipantHP(target, remainingHP)
		}

		m.appendCombatLog(fmt.Sprintf("%s deals %d damage to %s.", attacker, amount, target))

	case "PLAYER_DIED", "NPC_DIED":
		if !m.world.Combat.Active {
			return nil
		}

		fields := parseKeyValuesOrdered(body, "participant", "killer")
		participant := fields["participant"]
		killer := fields["killer"]

		m.updateCombatParticipantHP(participant, 0)
		if killer != "" {
			m.appendCombatLog(fmt.Sprintf("%s was defeated by %s.", participant, killer))
		} else {
			m.appendCombatLog(fmt.Sprintf("%s was defeated.", participant))
		}

	case "PLAYER_FLED":
		if !m.world.Combat.Active {
			return nil
		}

		fields := parseKeyValuesOrdered(body, "player", "damage", "remaining_hp")
		player := fields["player"]
		damage, _ := strconv.Atoi(fields["damage"])
		remainingHP, hasRemaining := strconv.Atoi(fields["remaining_hp"])

		for i := range m.world.Combat.Participants {
			p := &m.world.Combat.Participants[i]
			if p.IsPlayer && (strings.EqualFold(p.Name, player) || strings.EqualFold(p.ID, player)) {
				p.Escaped = true
				if hasRemaining == nil {
					p.HP = remainingHP
				}
				break
			}
		}

		if strings.EqualFold(player, m.player.Username) && hasRemaining == nil {
			m.setCombatPlayerHealth(remainingHP)
		}

		m.appendCombatLog(fmt.Sprintf("%s fled from combat (-%d HP).", player, damage))

	case "PLAYER_JOINED":
		if !m.world.Combat.Active {
			return nil
		}

		fields := parseKeyValuesOrdered(body, "player", "combat", "hp", "max_hp")
		player := fields["player"]
		if player == "" {
			return nil
		}

		hp, hasHP := strconv.Atoi(fields["hp"])
		maxHP, hasMaxHP := strconv.Atoi(fields["max_hp"])

		exists := false
		for i := range m.world.Combat.Participants {
			p := &m.world.Combat.Participants[i]
			if p.IsPlayer && (strings.EqualFold(p.Name, player) || strings.EqualFold(p.ID, player)) {
				exists = true
				if hasHP == nil {
					p.HP = hp
				}
				if hasMaxHP == nil && maxHP > 0 {
					p.MaxHP = maxHP
				}
				break
			}
		}
		if !exists {
			participant := CombatParticipantState{
				ID:        player,
				Name:      player,
				IsPlayer:  true,
				Connected: true,
				Alive:     true,
			}
			if hasHP == nil {
				participant.HP = hp
			}
			if hasMaxHP == nil && maxHP > 0 {
				participant.MaxHP = maxHP
			}
			m.world.Combat.Participants = append(m.world.Combat.Participants, participant)
		}

		m.appendCombatLog(fmt.Sprintf("%s joined the combat (enters next round).", player))

	case "HELP_REQUESTED":
		fields := parseKeyValuesOrdered(body, "player", "combat", "room")
		player := fields["player"]
		combatID := fields["combat"]
		room := fields["room"]
		m.appendCombatLog(fmt.Sprintf("%s requested combat assistance (%s) in %s.", player, combatID, room))

	case "ENDED":
		header, rewardsJSON, hasRewards := strings.Cut(body, " rewards=")
		headerFields := parsePayloadFields(header)
		combatID := headerFields["combat"]

		if combatID != "" && m.world.Combat.ID != "" && combatID != m.world.Combat.ID {
			return nil
		}

		m.world.Combat.Active = false
		m.world.Combat.AvailableActions = 0
		m.world.Combat.CurrentTurn = ""
		m.appendCombatLog("Combat ended.")

		if hasRewards {
			var rewards []struct {
				Items []struct {
					Name string `json:"name"`
				} `json:"Items"`
			}

			if err := json.Unmarshal([]byte(rewardsJSON), &rewards); err == nil {
				for _, reward := range rewards {
					for _, item := range reward.Items {
						message := "Reward received: " + item.Name
						m.appendCombatLog(message)
						m.appendEventLog(message)
					}
				}
			}
		}

		m.ui.PendingRefresh = RefreshStatus
		return sendCommand(m.client, "STATUS")
	}

	return nil
}

func (m *Model) handleRespawnEvent(payload string) tea.Cmd {
	fields := parseKeyValuesOrdered(payload, "player", "room", "hp", "max_hp", "combat")
	player := fields["player"]
	room := fields["room"]

	if player != "" && !strings.EqualFold(player, m.player.Username) {
		return nil
	}

	if maxHP, err := strconv.Atoi(fields["max_hp"]); err == nil && maxHP > 0 {
		m.player.Status.MaxHP = maxHP
	}
	if hp, err := strconv.Atoi(fields["hp"]); err == nil {
		m.setCombatPlayerHealth(hp)
	}

	m.communication.StatusMessage = "You respawned in " + room
	m.appendCombatLog("You respawned in " + room + ".")

	m.ui.PendingRefresh = RefreshRoom
	return sendCommand(m.client, "LOOK")
}

func (m *Model) updateCombatParticipantHP(targetName string, hp int) {
	if hp < 0 {
		hp = 0
	}

	if strings.EqualFold(targetName, m.player.Username) {
		m.setCombatPlayerHealth(hp)
		return
	}

	for i := range m.world.Combat.Participants {
		p := &m.world.Combat.Participants[i]
		if strings.EqualFold(p.Name, targetName) || strings.EqualFold(p.ID, targetName) {
			p.HP = hp
			p.Alive = hp > 0
			break
		}
	}
}
