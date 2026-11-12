package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"the-answer-protocol/internal/protocol"
)

const HandshakeTimeout = 10 * time.Second

var (
	ErrInvalidGreeting = errors.New(
		"invalid TAP server greeting",
	)

	ErrUnsupportedProtocol = errors.New(
		"unsupported TAP protocol version",
	)

	ErrNameInUse = errors.New(
		"username is already in use",
	)

	ErrAuthentication = errors.New(
		"authentication failed",
	)

	ErrConnectionAuthenticated = errors.New(
		"connection is already authenticated",
	)
)

type Connection struct {
	transport *Transport

	// Prevents concurrent authentication attempts.
	authMu sync.Mutex

	// Protected by authMu.
	authenticated bool
}

// Transport returns the underlying transport.
func (c *Connection) Transport() *Transport {
	if c == nil {
		return nil
	}

	return c.transport
}

// Dial establishes a TCP connection and validates the server greeting.
func Dial(ctx context.Context, address string) (*Connection, error) {
	if ctx == nil {
		return nil, errors.New("context cannot be nil")
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}

	transport := NewTransport(conn)
	connection := &Connection{
		transport: transport,
	}

	if err := ctx.Err(); err != nil {
		_ = transport.Close()
		return nil, err
	}

	if err := connection.receiveGreeting(); err != nil {
		_ = transport.Close()

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		return nil, err
	}

	if err := ctx.Err(); err != nil {
		_ = transport.Close()
		return nil, err
	}

	return connection, nil
}

// receiveGreeting validates the initial TAP server greeting.
func (c *Connection) receiveGreeting() (retErr error) {
	if c == nil || c.transport == nil {
		return fmt.Errorf("receive greeting: %w", net.ErrClosed)
	}

	if err := c.transport.SetReadDeadline(
		time.Now().Add(HandshakeTimeout),
	); err != nil {
		return fmt.Errorf("set greeting deadline: %w", err)
	}

	defer func() {
		if err := c.transport.SetReadDeadline(time.Time{}); err != nil {
			_ = c.transport.Close()

			if retErr == nil {
				retErr = fmt.Errorf("reset deadline: %w", err)
			}
		}
	}()

	message, err := c.transport.ReadLine()
	if err != nil {
		return fmt.Errorf("read server greeting: %w", err)
	}

	if message != "OK hello proto=1" {
		if strings.HasPrefix(message, "OK hello proto=") {
			return fmt.Errorf(
				"%w: received %q",
				ErrUnsupportedProtocol,
				message,
			)
		}

		return fmt.Errorf("%w: %q", ErrInvalidGreeting, message)
	}

	return nil
}

// Authenticate authenticates the client using the TAP CONNECT command.
func (c *Connection) Authenticate(username string) (retErr error) {
	if c == nil || c.transport == nil {
		return net.ErrClosed
	}

	c.authMu.Lock()
	defer c.authMu.Unlock()

	if c.authenticated {
		return ErrConnectionAuthenticated
	}

	if !protocol.ValidUsername(username) {
		return protocol.ErrInvalidUsername
	}

	if err := c.transport.SetReadDeadline(
		time.Now().Add(HandshakeTimeout),
	); err != nil {
		return fmt.Errorf("set authentication deadline: %w", err)
	}

	defer func() {
		if err := c.transport.SetReadDeadline(time.Time{}); err != nil {
			_ = c.transport.Close()

			if retErr == nil {
				retErr = fmt.Errorf("reset deadline: %w", err)
			}
		}
	}()

	if err := c.transport.WriteLine("CONNECT " + username); err != nil {
		_ = c.transport.Close()
		return fmt.Errorf("send CONNECT: %w", err)
	}

	message, err := c.transport.ReadLine()
	if err != nil {
		_ = c.transport.Close()
		return fmt.Errorf("read authentication response: %w", err)
	}

	switch {
	case message == "OK connected":
		c.authenticated = true
		return nil

	case message == "ERR 201 NAME_IN_USE":
		return ErrNameInUse

	case strings.HasPrefix(message, "ERR "):
		_ = c.transport.Close()
		return fmt.Errorf("%w: %s", ErrAuthentication, message)

	default:
		_ = c.transport.Close()
		return fmt.Errorf(
			"unexpected authentication response: %q",
			message,
		)
	}
}

// Close closes the underlying TCP connection.
func (c *Connection) Close() error {
	if c == nil || c.transport == nil {
		return nil
	}

	return c.transport.Close()
}