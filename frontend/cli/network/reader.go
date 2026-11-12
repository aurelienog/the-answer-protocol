package network

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"the-answer-protocol/internal/protocol"
)

const EventBufferSize = 128

var (
	ErrResponseBufferFull = errors.New(
		"response channel is full",
	)

	ErrNilReaderDependency = errors.New(
		"reader dependency cannot be nil",
	)

	ErrUnexpectedMessageKind = errors.New(
		"unexpected TAP message kind",
	)
)

type Reader struct {
	transport *Transport

	responses    chan<- IncomingMessage
	events       chan<- IncomingMessage
	onFatalError func(error)

	startOnce sync.Once
}

// NewReader creates a reader with its required dependencies.
func NewReader(
	transport *Transport,
	responses chan<- IncomingMessage,
	events chan<- IncomingMessage,
	onFatalError func(error),
) (*Reader, error) {
	if transport == nil ||
		responses == nil ||
		events == nil ||
		onFatalError == nil {
		return nil, ErrNilReaderDependency
	}

	return &Reader{
		transport:    transport,
		responses:    responses,
		events:       events,
		onFatalError: onFatalError,
	}, nil
}

// Start starts the reader goroutine.
// It can only be started once.
func (r *Reader) Start(ctx context.Context) {
	if r == nil || ctx == nil {
		return
	}

	r.startOnce.Do(func() {
		go r.run(ctx)
	})
}

// run continuously reads and processes incoming TAP messages.
func (r *Reader) run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		raw, err := r.transport.ReadLine()
		if err != nil {
			if ctx.Err() == nil {
				r.reportError(
					fmt.Errorf("read TAP message: %w", err),
				)
			}
			return
		}

		message, err := protocol.ParseMessage(raw)
		if err != nil {
			r.reportError(
				fmt.Errorf("parse TAP message %q: %w", raw, err),
			)
			return
		}

		incoming := IncomingMessage{
			Message: message,
			Raw:     raw,
		}

		switch message.Kind {
		case protocol.KindOK, protocol.KindERR:
			if !r.dispatchResponse(ctx, incoming) {
				return
			}

		case protocol.KindEVT:
			r.dispatchEvent(incoming)

		default:
			r.reportError(
				fmt.Errorf(
					"%w: %d",
					ErrUnexpectedMessageKind,
					message.Kind,
				),
			)
			return
		}
	}
}

// dispatchResponse delivers a response.
// A full response channel is a fatal synchronization error.
func (r *Reader) dispatchResponse(
	ctx context.Context,
	incoming IncomingMessage,
) bool {
	if ctx.Err() != nil {
		return false
	}

	select {
	case r.responses <- incoming:
		return true

	default:
		r.reportError(ErrResponseBufferFull)
		return false
	}
}

// dispatchEvent delivers an event without blocking.
// Events are dropped if the event queue is full.
func (r *Reader) dispatchEvent(incoming IncomingMessage) {
	select {
	case r.events <- incoming:
	default:
		// Event queue full. Event dropped.
	}
}

// reportError reports a fatal reader error to the Client.
func (r *Reader) reportError(err error) {
	r.onFatalError(err)
}