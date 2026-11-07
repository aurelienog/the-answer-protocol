package abuse

import (
	"strings"
	"time"
)

// AllowCommand checks the general command limit and, for CHAT, its stricter limit.
func (guard *AbuseGuard) AllowCommand(playerName, line string) bool {
	if guard == nil {
		return true
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return true
	}

	now := guard.now()
	playerKey := strings.ToLower(strings.TrimSpace(playerName))
	commandKey := "command:" + playerKey
	isChat := strings.EqualFold(fields[0], "CHAT")
	chatKey := "chat:" + playerKey

	guard.mu.Lock()
	defer guard.mu.Unlock()

	commandEvents := recent(guard.rateEvents[commandKey], now.Add(-commandWindow))
	if len(commandEvents) >= maxCommandsPerPlayer {
		guard.rateEvents[commandKey] = commandEvents
		return false
	}
	if isChat {
		chatEvents := recent(guard.rateEvents[chatKey], now.Add(-chatWindow))
		if len(chatEvents) >= maxChatsPerPlayer {
			guard.rateEvents[commandKey] = append(commandEvents, now)
			guard.rateEvents[chatKey] = chatEvents
			return false
		}
		guard.rateEvents[chatKey] = append(chatEvents, now)
	}
	guard.rateEvents[commandKey] = append(commandEvents, now)
	return true
}

// AllowCommand checks the request policies through an admitted connection.
func (lease *ConnectionLease) AllowCommand(playerName, line string) bool {
	if lease == nil || lease.guard == nil {
		return true
	}
	return lease.guard.AllowCommand(playerName, line)
}

func recent(events []time.Time, since time.Time) []time.Time {
	first := 0
	for first < len(events) && !events[first].After(since) {
		first++
	}
	return events[first:]
}
