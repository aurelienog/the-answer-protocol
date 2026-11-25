package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
)

// Serve accepts connections until the context is cancelled or the listener
// fails, then closes active connections and waits for their handlers to exit.
func Serve(ctx context.Context, listener net.Listener, handle func(net.Conn)) error {
	stopListener := context.AfterFunc(ctx, func() {
		_ = listener.Close()
	})
	defer stopListener()
	defer listener.Close()

	var (
		mu       sync.Mutex
		active   = make(map[net.Conn]struct{})
		handlers sync.WaitGroup
		serveErr error
	)

	for ctx.Err() == nil {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() == nil && !errors.Is(err, net.ErrClosed) {
				serveErr = fmt.Errorf("accept TCP connection: %w", err)
			}
			break
		}
		if ctx.Err() != nil {
			_ = conn.Close()
			break
		}

		mu.Lock()
		active[conn] = struct{}{}
		handlers.Add(1)
		mu.Unlock()

		go func(conn net.Conn) {
			defer handlers.Done()
			defer func() {
				mu.Lock()
				delete(active, conn)
				mu.Unlock()
			}()
			handle(conn)
		}(conn)
	}

	mu.Lock()
	connections := make([]net.Conn, 0, len(active))
	for conn := range active {
		connections = append(connections, conn)
	}
	mu.Unlock()
	for _, conn := range connections {
		_ = conn.Close()
	}
	handlers.Wait()

	return serveErr
}
