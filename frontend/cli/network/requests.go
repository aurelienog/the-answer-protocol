package network

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"the-answer-protocol/internal/protocol"
)

var (
	ErrRequestManagerClosed = errors.New(
		"request manager is closed",
	)

	ErrUnexpectedResponse = errors.New(
		"unexpected message received as request response",
	)

	ErrNilResponseChannel = errors.New(
		"request manager has no response channel",
	)

	ErrNilRequestContext = errors.New(
		"request manager context cannot be nil",
	)
)

type RequestManager struct {
	responses <-chan IncomingMessage

	// Allows only one active request at a time.
	gate chan struct{}

	// Child context of the Client lifecycle context.
	ctx    context.Context
	cancel context.CancelFunc

	closeOnce sync.Once
}

// NewRequestManager creates a request manager.
//
// The manager's lifecycle is tied to the provided client context.
func NewRequestManager(
	ctx context.Context,
	responses <-chan IncomingMessage,
) (*RequestManager, error) {
	if ctx == nil {
		return nil, ErrNilRequestContext
	}

	if responses == nil {
		return nil, ErrNilResponseChannel
	}

	managerCtx, cancel := context.WithCancel(ctx)

	return &RequestManager{
		responses: responses,
		gate:      make(chan struct{}, 1),
		ctx:       managerCtx,
		cancel:    cancel,
	}, nil
}

// Do sends a request and waits for its server response.
//
// The caller context only controls waiting for the request turn.
// Once sending begins, the request cannot be individually cancelled.
//
// Closing the request manager interrupts an active request.
func (m *RequestManager) Do(
	ctx context.Context,
	send func(context.Context) error,
) (protocol.Message, error) {
	if m == nil {
		return protocol.Message{}, ErrRequestManagerClosed
	}

	if ctx == nil {
		return protocol.Message{}, errors.New(
			"request context is nil",
		)
	}

	if send == nil {
		return protocol.Message{}, errors.New(
			"request send function is nil",
		)
	}

	// Acquire exclusive request ownership.
	select {
	case m.gate <- struct{}{}:
		defer func() {
			<-m.gate
		}()

	case <-ctx.Done():
		return protocol.Message{}, ctx.Err()

	case <-m.ctx.Done():
		return protocol.Message{}, ErrRequestManagerClosed
	}

	// Check cancellation again in case multiple select cases
	// became ready at approximately the same time.
	if err := ctx.Err(); err != nil {
		return protocol.Message{}, err
	}

	if m.ctx.Err() != nil {
		return protocol.Message{}, ErrRequestManagerClosed
	}

	// From this point, the request cannot be individually cancelled.
	// The send function must return only after the TCP write completes.
	if err := send(m.ctx); err != nil {
		if m.ctx.Err() != nil {
			return protocol.Message{}, ErrRequestManagerClosed
		}

		return protocol.Message{}, fmt.Errorf(
			"send request: %w",
			err,
		)
	}

	// Wait for the server response or client shutdown.
	select {
	case incoming, ok := <-m.responses:
		if !ok {
			m.Close()
			return protocol.Message{}, ErrRequestManagerClosed
		}

		return handleResponse(incoming)

	case <-m.ctx.Done():
		return protocol.Message{}, ErrRequestManagerClosed
	}
}

// Close permanently shuts down the request manager.
func (m *RequestManager) Close() {
	if m == nil {
		return
	}

	m.closeOnce.Do(func() {
		m.cancel()
	})
}

// handleResponse interprets a server response.
func handleResponse(
	incoming IncomingMessage,
) (protocol.Message, error) {
	switch incoming.Message.Kind {
	case protocol.KindOK:
		return incoming.Message, nil

	case protocol.KindERR:
		return incoming.Message, &protocol.ServerError{
			Code: incoming.Message.Code,
			Name: incoming.Message.Name,
		}

	default:
		return protocol.Message{}, fmt.Errorf(
			"%w: received message kind %d",
			ErrUnexpectedResponse,
			incoming.Message.Kind,
		)
	}
}