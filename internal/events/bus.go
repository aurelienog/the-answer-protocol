package events

import "sync"

// Publisher publishes internal game events.
type Publisher interface {
	Publish(event any)
}

// Handler processes a published event.
type Handler func(event any)

// Bus distributes events to registered handlers.
type Bus struct {
	mu       sync.RWMutex
	handlers []Handler
}

// NewBus creates an empty event bus.
func NewBus() *Bus {
	return &Bus{}
}

// Subscribe registers a handler for published events.
func (bus *Bus) Subscribe(handler Handler) {
	if bus == nil || handler == nil {
		return
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.handlers = append(bus.handlers, handler)
}

// Publish sends an event to every registered handler.
func (bus *Bus) Publish(event any) {
	if bus == nil {
		return
	}

	bus.mu.RLock()
	handlers := append([]Handler(nil), bus.handlers...)
	bus.mu.RUnlock()

	for _, handler := range handlers {
		handler(event)
	}
}
