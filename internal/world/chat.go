package world

import (
    "fmt"
    "sort"
    "strings"
)

// SendChat validates a chat message and returns the event to publish.
func (w *World) SendChat(playerName, scope, message string) (ChatMessage, error) {
    if w == nil {
        return ChatMessage{}, fmt.Errorf("%w: world is nil", ErrInvalidWorldState)
    }

    playerName = strings.TrimSpace(playerName)
    scope = strings.ToUpper(strings.TrimSpace(scope))
    message = strings.TrimSpace(message)

    if playerName == "" || message == "" {
        return ChatMessage{}, fmt.Errorf("%w: invalid chat arguments", ErrInvalidArgument)
    }

    w.mu.RLock()
    defer w.mu.RUnlock()

    player, exists := w.players[playerName]
    if !exists {
        return ChatMessage{}, fmt.Errorf("%w: %q", ErrPlayerNotFound, playerName)
    }

    roomID := player.RoomID

    var recipients []string

    switch scope {
    case "GLOBAL":
        recipients = make([]string, 0, len(w.players))

        for name := range w.players {
            recipients = append(recipients, name)
        }

        sort.Strings(recipients)

    case "ROOM":
        recipients = w.getPlayersInRoomLocked(roomID)

    case "GROUP":
        groupID, exists := w.playerGroups[playerName]
        if !exists {
            return ChatMessage{}, fmt.Errorf(
                "%w: player %q is not in a group",
                ErrPlayerNotInGroup,
                playerName,
            )
        }

        group, exists := w.groups[groupID]
        if !exists {
            return ChatMessage{}, fmt.Errorf(
                "%w: group %q does not exist",
                ErrInvalidWorldState,
                groupID,
            )
        }

        recipients = cloneSlice(group.Members)

    default:
        return ChatMessage{}, fmt.Errorf(
            "%w: unsupported chat scope %q",
            ErrInvalidArgument,
            scope,
        )
    }

    return ChatMessage{
        Scope:      scope,
        PlayerName: playerName,
        Message:    message,
        Recipients: recipients,
    }, nil
}