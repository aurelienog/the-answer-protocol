package network

import (
	"context"
	"errors"
	"sync"
)

const OutboxCapacity = 64

var (
	ErrOutboxClosed = errors.New(
		"outbox is closed",
	)

	ErrNilOutboxContext = errors.New(
		"outbox context cannot be nil",
	)
)

// OutboundMessage represents a message waiting to be written.
//
// Result receives the result of Transport.WriteLine.
// Its buffer allows the Writer to report completion without
// blocking if the sender has not started waiting yet.
type OutboundMessage struct {
	Line   string
	Result chan error
}

// Outbox is a bounded FIFO queue for outgoing TAP messages.
//
// It supports concurrent producers and a single consumer.
type Outbox struct {
	mu sync.Mutex

	queue    []OutboundMessage
	capacity int
	closed   bool

	// Notifications for queue state changes.
	notEmpty chan struct{}
	notFull  chan struct{}

	// Closed when the Outbox shuts down.
	done chan struct{}

	closeOnce sync.Once
}

// NewOutbox creates an outgoing message queue.
func NewOutbox() *Outbox {
	return &Outbox{
		queue:    make([]OutboundMessage, 0, OutboxCapacity),
		capacity: OutboxCapacity,
		notEmpty: make(chan struct{}),
		notFull:  make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Enqueue adds a message to the queue.
//
// If the queue is full, it waits until space becomes available,
// the context is cancelled, or the Outbox is closed.
//
// The returned channel receives the result of the actual TCP write.
func (o *Outbox) Enqueue(
	ctx context.Context,
	line string,
) (<-chan error, error) {
	if o == nil {
		return nil, ErrOutboxClosed
	}

	if ctx == nil {
		return nil, ErrNilOutboxContext
	}

	if err := validateOutgoingMessage(line); err != nil {
		return nil, err
	}

	result := make(chan error, 1)

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		o.mu.Lock()

		if o.closed {
			o.mu.Unlock()
			return nil, ErrOutboxClosed
		}

		if err := ctx.Err(); err != nil {
			o.mu.Unlock()
			return nil, err
		}

		if len(o.queue) < o.capacity {
			o.queue = append(o.queue, OutboundMessage{
				Line:   line,
				Result: result,
			})

			// Notify the consumer that the queue may no longer be empty.
			close(o.notEmpty)
			o.notEmpty = make(chan struct{})

			o.mu.Unlock()

			return result, nil
		}

		// Capture the current notification channel while holding the mutex.
		wait := o.notFull

		o.mu.Unlock()

		select {
		case <-wait:
		case <-o.done:
			return nil, ErrOutboxClosed
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// Next retrieves the next message from the queue.
//
// It blocks until a message becomes available, the context is
// cancelled, or the Outbox is closed.
func (o *Outbox) Next(
	ctx context.Context,
) (OutboundMessage, error) {
	if o == nil {
		return OutboundMessage{}, ErrOutboxClosed
	}

	if ctx == nil {
		return OutboundMessage{}, ErrNilOutboxContext
	}

	for {
		if err := ctx.Err(); err != nil {
			return OutboundMessage{}, err
		}

		o.mu.Lock()

		if o.closed {
			o.mu.Unlock()
			return OutboundMessage{}, ErrOutboxClosed
		}

		if err := ctx.Err(); err != nil {
			o.mu.Unlock()
			return OutboundMessage{}, err
		}

		if len(o.queue) > 0 {
			message := o.queue[0]

			// Remove the first element while preserving FIFO order.
			copy(o.queue, o.queue[1:])
			o.queue[len(o.queue)-1] = OutboundMessage{}
			o.queue = o.queue[:len(o.queue)-1]

			// Notify waiting producers that capacity may be available.
			close(o.notFull)
			o.notFull = make(chan struct{})

			o.mu.Unlock()

			return message, nil
		}

		// Capture the current notification channel while holding the mutex.
		wait := o.notEmpty

		o.mu.Unlock()

		select {
		case <-wait:
		case <-o.done:
			return OutboundMessage{}, ErrOutboxClosed
		case <-ctx.Done():
			return OutboundMessage{}, ctx.Err()
		}
	}
}

// Close permanently shuts down the Outbox.
//
// Messages that have not yet been retrieved by the Writer
// receive ErrOutboxClosed as their write result.
func (o *Outbox) Close() {
	if o == nil {
		return
	}

	o.closeOnce.Do(func() {
		o.mu.Lock()
		defer o.mu.Unlock()

		o.closed = true

		for _, message := range o.queue {
			message.Result <- ErrOutboxClosed
		}

		o.queue = nil

		close(o.done)
	})
}