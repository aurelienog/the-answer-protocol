
package ui

import (
    "encoding/json"
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "the-answer-protocol/internal/protocol"
)

const maxEventLogEntries = 100

func (m Model) handleServerEvent(
    msg serverEventMsg,
) (tea.Model, tea.Cmd) {
    event := msg.event.Message

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

        if event.Kind == protocol.KindEVT && event.EventType == "COMBAT" {
            m.handleCombatEvent(event.Data)
        }
    }

    return m, waitForServerEvent(m.client)
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
    // STARTED no contiene JSON, por lo que podemos separar sus campos.
    parts := strings.Fields(data)
    if len(parts) == 0 {
        return
    }

    switch parts[0] {
    case "STARTED":
        var combatID string
        var participants []string

        for _, part := range parts[1:] {
            key, value, ok := strings.Cut(part, "=")
            if !ok {
                continue
            }

            switch key {
            case "combat":
                combatID = value
            case "participants":
                participants = strings.Split(value, ",")
            }
        }

        if combatID == "" {
            return
        }

        for _, participant := range participants {
            if participant == m.player.Username {
                m.world.InCombat = true
                m.world.CombatID = combatID
            }
        }

    case "ENDED":
        // Separamos la cabecera del JSON para no dividir
        // las descripciones de los objetos por sus espacios.
        header, rewardsJSON, hasRewards := strings.Cut(data, " rewards=")

        headerParts := strings.Fields(header)
        if len(headerParts) < 2 {
            return
        }

        var combatID string

        for _, part := range headerParts[1:] {
            key, value, ok := strings.Cut(part, "=")
            if ok && key == "combat" {
                combatID = value
                break
            }
        }

        // Ignoramos eventos de otros combates.
        if combatID == "" || combatID != m.world.CombatID {
            return
        }

        m.world.InCombat = false
        m.world.CombatID = ""

        if !hasRewards {
            return
        }

        // Solo necesitamos los nombres de los objetos para mostrarlos.
        var rewards []struct {
            Items []struct {
                Name string `json:"name"`
            } `json:"Items"`
        }

        if err := json.Unmarshal([]byte(rewardsJSON), &rewards); err != nil {
            m.appendEventLog(
                "No se pudieron leer las recompensas del combate.",
            )
            return
        }

        for _, reward := range rewards {
            for _, item := range reward.Items {
                m.appendEventLog("Recompensa recibida: " + item.Name)
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
