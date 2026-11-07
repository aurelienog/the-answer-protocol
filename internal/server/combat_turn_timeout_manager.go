package server

import (
	"context"
	"sync"
	"time"

	"the-answer-protocol/internal/world"
)

// CombatTurnTimeoutManager automatically skips player turns that exceed the
// configured duration.
type CombatTurnTimeoutManager struct {
	timeout   time.Duration
	onTimeout func(world.CombatTurnChanged)

	mu     sync.Mutex
	timers map[string]*time.Timer
	tokens map[string]uint64
	nextID uint64
}

func NewCombatTurnTimeoutManager(
	timeout time.Duration,
	onTimeout func(world.CombatTurnChanged),
) *CombatTurnTimeoutManager {
	return &CombatTurnTimeoutManager{
		timeout:   timeout,
		onTimeout: onTimeout,
		timers:    make(map[string]*time.Timer),
		tokens:    make(map[string]uint64),
	}
}

// Handle starts, replaces, or cancels a timer when the active participant
// changes.
func (manager *CombatTurnTimeoutManager) Handle(event any) {
	turn, ok := event.(world.CombatTurnChanged)
	if manager == nil {
		return
	}
	if !ok {
		var ended world.CombatEnded
		switch value := event.(type) {
		case world.CombatEnded:
			ended = value
		case *world.CombatEnded:
			if value == nil {
				return
			}
			ended = *value
		default:
			return
		}
		if ended.CombatID != "" {
			manager.cancel(ended.CombatID)
		}
		return
	}
	if manager.timeout <= 0 || manager.onTimeout == nil {
		return
	}

	manager.mu.Lock()
	if timer := manager.timers[turn.CombatID]; timer != nil {
		timer.Stop()
		delete(manager.timers, turn.CombatID)
		delete(manager.tokens, turn.CombatID)
	}
	if turn.IsPlayer {
		manager.nextID++
		token := manager.nextID
		manager.tokens[turn.CombatID] = token
		manager.timers[turn.CombatID] = time.AfterFunc(
			manager.timeout,
			func() { manager.expire(turn, token) },
		)
	}
	manager.mu.Unlock()
}

func (manager *CombatTurnTimeoutManager) cancel(combatID string) {
	manager.mu.Lock()
	if timer := manager.timers[combatID]; timer != nil {
		timer.Stop()
		delete(manager.timers, combatID)
		delete(manager.tokens, combatID)
	}
	manager.mu.Unlock()
}

func (manager *CombatTurnTimeoutManager) expire(
	turn world.CombatTurnChanged,
	token uint64,
) {
	manager.mu.Lock()
	if manager.tokens[turn.CombatID] != token {
		manager.mu.Unlock()
		return
	}
	delete(manager.timers, turn.CombatID)
	delete(manager.tokens, turn.CombatID)
	manager.mu.Unlock()

	manager.onTimeout(turn)
}

// Stop cancels all pending turn timers.
func (manager *CombatTurnTimeoutManager) Stop() {
	if manager == nil {
		return
	}
	manager.mu.Lock()
	for combatID, timer := range manager.timers {
		timer.Stop()
		delete(manager.timers, combatID)
		delete(manager.tokens, combatID)
	}
	manager.mu.Unlock()
}

// Run stops all timers when the owning server context is cancelled.
func (manager *CombatTurnTimeoutManager) Run(ctx context.Context) {
	if manager == nil || ctx == nil {
		return
	}
	<-ctx.Done()
	manager.Stop()
}
