package abuse

import (
	"context"
	"time"
)

// RunCleanup periodically removes expired IP and rate-limit records.
func (guard *AbuseGuard) RunCleanup(ctx context.Context, interval time.Duration) {
	if guard == nil {
		return
	}
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			guard.CleanupExpired(now)
		}
	}
}

// CleanupExpired removes inactive records whose windows have elapsed.
func (guard *AbuseGuard) CleanupExpired(now time.Time) {
	if guard == nil {
		return
	}
	guard.mu.Lock()
	defer guard.mu.Unlock()

	attemptCutoff := now.Add(-guard.config.ConnectionAttemptWindow)
	for ip, state := range guard.ipStates {
		state.attempts = recent(state.attempts, attemptCutoff)
		if state.activeConnections == 0 && len(state.attempts) == 0 {
			delete(guard.ipStates, ip)
		}
	}
	for key, events := range guard.rateEvents {
		cutoff := now.Add(-commandWindow)
		if len(key) >= 5 && key[:5] == "chat:" {
			cutoff = now.Add(-chatWindow)
		}
		events = recent(events, cutoff)
		if len(events) == 0 {
			delete(guard.rateEvents, key)
		} else {
			guard.rateEvents[key] = events
		}
	}
}
