package server

import (
	"context"
	"sync"
	"time"

	"the-answer-protocol/internal/world"
)

// NPCRespawnManager restores defeated NPCs when their respawn deadline expires.
type NPCRespawnManager struct {
	onRespawn func(world.NPCRespawnScheduled)

	mu     sync.Mutex
	timers map[string]*time.Timer
	tokens map[string]uint64
	nextID uint64
}

func NewNPCRespawnManager(
	onRespawn func(world.NPCRespawnScheduled),
) *NPCRespawnManager {
	return &NPCRespawnManager{
		onRespawn: onRespawn,
		timers:    make(map[string]*time.Timer),
		tokens:    make(map[string]uint64),
	}
}

// Handle starts or replaces a timer for a scheduled NPC respawn.
func (manager *NPCRespawnManager) Handle(event any) {
	scheduled, ok := event.(world.NPCRespawnScheduled)
	if !ok {
		if value, isPointer := event.(*world.NPCRespawnScheduled); isPointer && value != nil {
			scheduled = *value
			ok = true
		}
	}
	if manager == nil || !ok || manager.onRespawn == nil ||
		scheduled.NPCID == "" || scheduled.RespawnAt.IsZero() {
		return
	}

	manager.mu.Lock()
	if timer := manager.timers[scheduled.NPCID]; timer != nil {
		timer.Stop()
		delete(manager.timers, scheduled.NPCID)
		delete(manager.tokens, scheduled.NPCID)
	}

	manager.nextID++
	token := manager.nextID
	manager.tokens[scheduled.NPCID] = token
	delay := time.Until(scheduled.RespawnAt)
	if delay < 0 {
		delay = 0
	}
	manager.timers[scheduled.NPCID] = time.AfterFunc(
		delay,
		func() { manager.expire(scheduled, token) },
	)
	manager.mu.Unlock()
}

func (manager *NPCRespawnManager) expire(
	scheduled world.NPCRespawnScheduled,
	token uint64,
) {
	manager.mu.Lock()
	if manager.tokens[scheduled.NPCID] != token {
		manager.mu.Unlock()
		return
	}
	delete(manager.timers, scheduled.NPCID)
	delete(manager.tokens, scheduled.NPCID)
	manager.mu.Unlock()

	manager.onRespawn(scheduled)
}

// Stop cancels all pending NPC respawn timers.
func (manager *NPCRespawnManager) Stop() {
	if manager == nil {
		return
	}
	manager.mu.Lock()
	for npcID, timer := range manager.timers {
		timer.Stop()
		delete(manager.timers, npcID)
		delete(manager.tokens, npcID)
	}
	manager.mu.Unlock()
}

// Run stops all timers when the owning server context is cancelled.
func (manager *NPCRespawnManager) Run(ctx context.Context) {
	if manager == nil || ctx == nil {
		return
	}
	<-ctx.Done()
	manager.Stop()
}
