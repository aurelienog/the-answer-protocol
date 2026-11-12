package network

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrWriterClosed = errors.New(
		"writer is closed",
	)

	ErrNilWriterDependency = errors.New(
		"writer dependency cannot be nil",
	)

	ErrNilWriteResult = errors.New(
		"outbox returned a nil write result channel",
	)
)

type Writer struct {
	transport    *Transport
	outbox       *Outbox
	onFatalError func(error)

	startOnce sync.Once
}

// NewWriter creates a writer for the given transport and outbox.
func NewWriter(
	transport *Transport,
	outbox *Outbox,
	onFatalError func(error),
) (*Writer, error) {
	if transport == nil ||
		outbox == nil ||
		onFatalError == nil {
		return nil, ErrNilWriterDependency
	}

	return &Writer{
		transport:    transport,
		outbox:       outbox,
		onFatalError: onFatalError,
	}, nil
}

// Start starts the writer goroutine.
// Only one writer goroutine can be started.
func (w *Writer) Start(ctx context.Context) {
	if w == nil || ctx == nil {
		return
	}

	w.startOnce.Do(func() {
		go w.run(ctx)
	})
}

// Send enqueues a TAP message and waits until it has been written.
func (w *Writer) Send(
	ctx context.Context,
	message string,
) error {
	if w == nil {
		return ErrWriterClosed
	}

	if ctx == nil {
		return errors.New("writer context cannot be nil")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	result, err := w.outbox.Enqueue(ctx, message)
	if err != nil {
		return fmt.Errorf("enqueue TAP message: %w", err)
	}

	if result == nil {
		return ErrNilWriteResult
	}

	select {
	case err := <-result:
		if err != nil {
			return fmt.Errorf("write TAP message: %w", err)
		}

		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

// run consumes outgoing messages and writes them to TCP.
func (w *Writer) run(ctx context.Context) {
	for {
		outgoing, err := w.outbox.Next(ctx)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, ErrOutboxClosed) {
				return
			}

			w.reportError(
				fmt.Errorf("get outgoing message: %w", err),
			)
			return
		}

		if ctx.Err() != nil {
			outgoing.Result <- ctx.Err()
			return
		}

		err = w.transport.WriteLine(outgoing.Line)

		// Notify the sender that the write has completed.
		outgoing.Result <- err

		if err != nil {
			w.reportError(
				fmt.Errorf("writer: %w", err),
			)
			return
		}
	}
}

// reportError reports a fatal writer error to Client.
func (w *Writer) reportError(err error) {
	w.onFatalError(err)
}