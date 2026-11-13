package network

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"log"
	"the-answer-protocol/internal/protocol"
)

var (
	ErrClientClosed = errors.New(
		"client is closed",
	)

	ErrAlreadyAuthenticated = errors.New(
		"client is already authenticated",
	)

	ErrNilClientContext = errors.New(
		"client context cannot be nil",
	)

	ErrNilClientConnection = errors.New(
		"client connection cannot be nil",
	)

	ErrClientNotAuthenticated = errors.New(
		"client is not authenticated",
	)
)

const ResponseBufferSize = 1

type clientState uint8

const (
	stateConnected clientState = iota
	stateAuthenticating
	stateAuthenticated
	stateClosed
)

type Client struct {
	mu     sync.RWMutex
	authMu sync.Mutex

	connection *Connection

	ctx        context.Context
	cancel     context.CancelFunc

	reader *Reader
	events chan IncomingMessage

	outbox *Outbox
	writer *Writer

	requestManager *RequestManager

	state clientState

	closeOnce sync.Once
	closeErr  error
}

// NewClient creates a client using an established TCP connection.
// Authentication is performed separately through Client.Authenticate.
func NewClient(
	parentCtx context.Context,
	connection *Connection,
) (*Client, error) {
	if parentCtx == nil {
		return nil, ErrNilClientContext
	}

	if connection == nil {
		return nil, ErrNilClientConnection
	}

	if err := parentCtx.Err(); err != nil {
		_ = connection.Close()
		return nil, err
	}

	ctx, cancel := context.WithCancel(parentCtx)

	responses := make(
		chan IncomingMessage,
		ResponseBufferSize,
	)

	events := make(
		chan IncomingMessage,
		EventBufferSize,
	)

	outbox := NewOutbox()

	requestManager, err := NewRequestManager(ctx, responses)
	if err != nil {
		cancel()
		outbox.Close()
		connection.Close()

		return nil, fmt.Errorf(
			"create request manager: %w",
			err,
		)
	}

	client := &Client{
		connection:     connection,
		ctx:            ctx,
		cancel:         cancel,
		events:         events,
		outbox:         outbox,
		requestManager: requestManager,
		state:          stateConnected,
	}

	// Reader and Writer report fatal errors directly to the Client.
	onFatalError := func(err error) {
		log.Printf("Client fatal network error: %v", err)
		client.initiateClose(err)
	}

	reader, err := NewReader(
		connection.Transport(),
		responses,
		events,
		onFatalError,
	)
	if err != nil {
		cancel()
		requestManager.Close()
		outbox.Close()
		connection.Close()

		return nil, fmt.Errorf(
			"create reader: %w",
			err,
		)
	}

	writer, err := NewWriter(
		connection.Transport(),
		outbox,
		onFatalError,
	)
	if err != nil {
		cancel()
		requestManager.Close()
		outbox.Close()
		connection.Close()

		return nil, fmt.Errorf(
			"create writer: %w",
			err,
		)
	}

	client.reader = reader
	client.writer = writer

	return client, nil
}

// Authenticate authenticates the client with the TAP server.
// Reader and Writer start only after successful authentication.
func (c *Client) Authenticate(username string) error {
	if c == nil {
		return ErrClientClosed
	}

	c.authMu.Lock()
	defer c.authMu.Unlock()

	c.mu.Lock()

	switch c.state {
	case stateClosed:
		c.mu.Unlock()
		return ErrClientClosed

	case stateAuthenticated:
		c.mu.Unlock()
		return ErrAlreadyAuthenticated
	}

	c.state = stateAuthenticating
	c.mu.Unlock()

	if err := c.ctx.Err(); err != nil {
		c.initiateClose(err)
		return ErrClientClosed
	}

	if err := c.connection.Authenticate(username); err != nil {
		c.mu.RLock()
		closed := c.state == stateClosed
		c.mu.RUnlock()

		if closed {
			return ErrClientClosed
		}

		if errors.Is(err, ErrNameInUse) ||
			errors.Is(err, protocol.ErrInvalidUsername) {

			c.mu.Lock()

			if c.state != stateClosed {
				c.state = stateConnected
			}

			c.mu.Unlock()

			return fmt.Errorf("authenticate client: %w", err)
		}

		c.initiateClose(err)

		return fmt.Errorf("authenticate client: %w", err)
	}

	c.mu.Lock()

	if c.state == stateClosed {
		c.mu.Unlock()
		return ErrClientClosed
	}

	c.state = stateAuthenticated

	// Start while holding the state lock so Close cannot race
	// between the state transition and starting the pumps.
	c.writer.Start(c.ctx)
	c.reader.Start(c.ctx)

	c.mu.Unlock()

	return nil
}

// Send sends one TAP command and waits for its response.
// Only one request can be in flight because TAP has no request IDs.
func (c *Client) Send(
	ctx context.Context,
	command string,
) (protocol.Message, error) {
	if c == nil {
		return protocol.Message{}, ErrClientClosed
	}

	if ctx == nil {
		return protocol.Message{}, ErrNilClientContext
	}

	c.mu.RLock()
	state := c.state
	c.mu.RUnlock()

	if state == stateClosed || c.ctx.Err() != nil {
		return protocol.Message{}, ErrClientClosed
	}

	if state != stateAuthenticated {
		return protocol.Message{}, ErrClientNotAuthenticated
	}

	return c.requestManager.Do(
		ctx,
		func(sendCtx context.Context) error {
			return c.writer.Send(sendCtx, command)
		},
	)
}

// Events returns the channel containing asynchronous server events.
// The channel is not closed by Client.Close.
// Use Done to detect client shutdown.
func (c *Client) Events() <-chan IncomingMessage {
	if c == nil {
		return nil
	}

	return c.events
}

// Done returns a channel closed when the client shuts down.
func (c *Client) Done() <-chan struct{} {
	if c == nil {
		return nil
	}

	return c.ctx.Done()
}

// initiateClose shuts down the client exactly once.
// It does not wait for Reader or Writer.
func (c *Client) initiateClose(cause error) {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.state = stateClosed

		c.mu.Unlock()

		c.cancel()
		c.requestManager.Close()
		c.outbox.Close()

		connectionErr := c.connection.Close()

		if cause != nil && connectionErr != nil {
			c.closeErr = fmt.Errorf(
				"%w (also failed to close connection: %v)",
				cause,
				connectionErr,
			)
		} else if cause != nil {
			c.closeErr = cause
		} else {
			c.closeErr = connectionErr
		}
	})
}

// Close shuts down the client.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}

	c.initiateClose(nil)

	return c.closeErr
}