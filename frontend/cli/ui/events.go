package ui

import (
    "encoding/json"
    "strings"
    "strconv"
    "fmt"

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
                m.communication.ChatLog = m.communication.ChatLog[
                    len(m.communication.ChatLog)-maxEventLogEntries:
                ]
            }
        }
    } else {
        m.appendEventLog(msg.event.Raw)

        if event.Kind == protocol.KindEVT {
            switch event.EventType {
            case "COMBAT":
                m.handleCombatEvent(event.Data)
            
                if m.world.Combat.Active &&
                    m.navigation.CurrentView != ViewCombat {
                    cmd = m.startViewTransition(ViewCombat)
                }

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

func (m *Model) handleCombatEvent(data string) {
    if strings.HasPrefix(data, "STARTED ") {
        m.handleCombatStartedEvent(strings.TrimPrefix(data, "STARTED "))
        return
    }

    parts := strings.Fields(data)
    if len(parts) == 0 {
        return
    }

    fields := parseCombatFields(parts[1:])

    switch parts[0] {
    case "TURN":
        if !m.world.Combat.Active {
            return
        }

        m.world.Combat.CurrentTurn = fields["participant"]

        if round, err := strconv.Atoi(fields["round"]); err == nil {
            m.world.Combat.Round = round
        }

        if actions, err := strconv.Atoi(fields["actions"]); err == nil {
            m.world.Combat.AvailableActions = actions
        }

        m.appendCombatLog("Turn: " + m.world.Combat.CurrentTurn)

    case "ATTACK":
        if !m.world.Combat.Active {
            return
        }

        fields := parseCombatFields(parts[1:])

        attacker := fields["attacker"]
        target := fields["target"]
        attackerType := fields["attacker_type"]

        attackerIsPlayer := attackerType == "PLAYER"

        m.ensureCombatParticipant(attacker, attackerIsPlayer)
        m.ensureCombatParticipant(target, !attackerIsPlayer)

        m.appendCombatLog(
            attacker + " attacks " + target + ".",
        )

    case "DAMAGE":
        if !m.world.Combat.Active {
            return
        }

        fields := parseCombatFields(parts[1:])

        attacker := fields["attacker"]
        target := fields["target"]

        amount, _ := strconv.Atoi(fields["amount"])
        remainingHP, _ := strconv.Atoi(fields["remaining_hp"])
        m.appendEventLog(
            fmt.Sprintf(
                "DEBUG DAMAGE: target=%s remainingHP=%d",
                target,
                remainingHP,
            ),
        )
        targetIsPlayer := false

        for _, existing := range m.world.Combat.Participants {
            if existing.ID == target || existing.Name == target {
                targetIsPlayer = existing.IsPlayer
                break
            }
        }

        participant := m.ensureCombatParticipant(target, targetIsPlayer)

        if participant != nil {
            participant.HP = remainingHP

            if remainingHP <= 0 {
                participant.Alive = false
            }
        }

        m.appendCombatLog(
            attacker + " deals " +
                strconv.Itoa(amount) + " damage to " + target + ".",
        )

    case "PLAYER_JOINED":
        if !m.world.Combat.Active {
            return
        }

        fields := parseCombatFields(parts[1:])
        player := fields["player"]

        if player == "" {
            return
        }

        m.ensureCombatParticipant(player, true)
        m.appendCombatLog(player + " joined the combat.")

    case "PLAYER_FLED":
        if !m.world.Combat.Active {
            return
        }

        fields := parseCombatFields(parts[1:])

        player := fields["player"]
        remainingHP, _ := strconv.Atoi(fields["remaining_hp"])

        participant := m.ensureCombatParticipant(player, true)

        if participant != nil {
            participant.HP = remainingHP
            participant.Escaped = true
        }

        m.appendCombatLog(player + " fled from combat.")

    case "PLAYER_DIED", "NPC_DIED":
        if !m.world.Combat.Active {
            return
        }

        fields := parseCombatFields(parts[1:])

        name := fields["participant"]
        isPlayer := parts[0] == "PLAYER_DIED"

        participant := m.ensureCombatParticipant(name, isPlayer)

        if participant != nil {
            participant.Alive = false
            participant.HP = 0
        }

        m.appendCombatLog(name + " was defeated.")

    case "ENDED":
        header, rewardsJSON, hasRewards := strings.Cut(
            data,
            " rewards=",
        )

        headerParts := strings.Fields(header)
        fields := parseCombatFields(headerParts[1:])

        combatID := fields["combat"]

        if combatID == "" || combatID != m.world.Combat.ID {
            return
        }

        m.world.Combat.Active = false
        m.world.Combat.AvailableActions = 0
        m.world.Combat.CurrentTurn = ""

        m.appendCombatLog("Combat ended.")

        if !hasRewards {
            return
        }

        var rewards []struct {
            Items []struct {
                Name string `json:"name"`
            } `json:"Items"`
        }

        if err := json.Unmarshal([]byte(rewardsJSON), &rewards); err != nil {
            m.appendCombatLog("Unable to read combat rewards.")
            return
        }

        for _, reward := range rewards {
            for _, item := range reward.Items {
                message := "Reward received: " + item.Name
                m.appendCombatLog(message)
                m.appendEventLog(message)
            }
        }
    }
}

func (m *Model) appendEventLog(message string) {
    m.communication.EventLog = append(
        m.communication.EventLog,
        message,
    )

    if len(m.communication.EventLog) > maxEventLogEntries {
        m.communication.EventLog = m.communication.EventLog[
            len(m.communication.EventLog)-maxEventLogEntries:
        ]
    }
}

func (m *Model) handleGroupEvent(data string) tea.Cmd {
    if m.navigation.CurrentView != ViewGroup {
        return nil
    }

    parts := strings.Fields(data)
    if len(parts) == 0 {
        return nil
    }

    switch parts[0] {
    case "JOIN", "LEAVE":
        // Refresh group information when another member joins or leaves.
        m.ui.PendingRefresh = RefreshGroup
        return sendCommand(m.client, "GROUP INFO")

    default:
        return nil
    }
}

func parseCombatFields(parts []string) map[string]string {
    fields := make(map[string]string)
    currentKey := ""

    for _, part := range parts {
        key, value, ok := strings.Cut(part, "=")

        if ok {
            currentKey = key
            fields[currentKey] = value
            continue
        }

        if currentKey != "" {
            fields[currentKey] += " " + part
        }
    }

    return fields
}

func (m *Model) ensureCombatParticipant(
    name string,
    isPlayer bool,
) *CombatParticipantState {
    if name == "" {
        return nil
    }

    for i := range m.world.Combat.Participants {
        participant := &m.world.Combat.Participants[i]

        if participant.ID == name || participant.Name == name {
            return participant
        }
    }

    m.world.Combat.Participants = append(
        m.world.Combat.Participants,
        CombatParticipantState{
            ID:        name,
            Name:      name,
            IsPlayer:  isPlayer,
            Alive:     true,
            Connected: true,
        },
    )

    return &m.world.Combat.Participants[len(m.world.Combat.Participants)-1]
}

func (m *Model) appendCombatLog(message string) {
    m.world.Combat.Log = append(
        m.world.Combat.Log,
        message,
    )

    if len(m.world.Combat.Log) > maxEventLogEntries {
        m.world.Combat.Log = m.world.Combat.Log[
            len(m.world.Combat.Log)-maxEventLogEntries:
        ]
    }
}

func (m *Model) handleCombatStartedEvent(data string) {
    header, participantsJSON, ok := strings.Cut(data, " participants=")
    if !ok {
        return
    }

    fields := parseCombatFields(strings.Fields(header))
    combatID := fields["combat"]

    if combatID == "" {
        return
    }

    var participants []CombatParticipantState

    if err := json.Unmarshal([]byte(participantsJSON), &participants); err != nil {
        m.appendEventLog("Unable to read combat participants.")
        return
    }

    isParticipant := false

    for i := range participants {
        participant := &participants[i]

        if participant.IsPlayer {
            participant.Connected = true

            if participant.ID == m.player.Username ||
                participant.Name == m.player.Username {
                isParticipant = true
            }
        }
    }

    // Ignore combat events if the local player is not a participant.
    if !isParticipant {
        return
    }

    m.world.Combat = CombatState{
        ID:           combatID,
        Active:       true,
        Round:        1,
        Participants: participants,
        Log:          []string{"Combat started."},
    }
}