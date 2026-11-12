package network_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"the-answer-protocol/frontend/cli/network"
	"the-answer-protocol/internal/protocol"
)

func startTCPServer(t *testing.T, handler func(net.Conn)) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		handler(conn)
	}()
	return listener.Addr().String()
}

func writeTCPLine(conn net.Conn, line string) {
	_, _ = fmt.Fprintln(conn, line)
}

func readTCPLine(conn net.Conn) (string, error) {
	line, err := bufio.NewReader(conn).ReadString('\n')
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), err
}

func TestDialGreetingValidation(t *testing.T) {
	tests := []struct {
		name     string
		greeting string
		wantErr  error
	}{
		{name: "valid greeting", greeting: "OK hello proto=1"},
		{
			name:     "incompatible protocol version",
			greeting: "OK hello proto=2",
			wantErr:  network.ErrUnsupportedProtocol,
		},
		{
			name:     "invalid greeting",
			greeting: "WELCOME",
			wantErr:  network.ErrInvalidGreeting,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			address := startTCPServer(t, func(conn net.Conn) {
				writeTCPLine(conn, tt.greeting)
			})

			connection, err := network.Dial(context.Background(), address)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Dial() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Dial() error = %v", err)
			}
			if err := connection.Close(); err != nil {
				t.Errorf("Close() error = %v", err)
			}
		})
	}
}

func TestDialGreetingTimeout(t *testing.T) {
	address := startTCPServer(t, func(conn net.Conn) {
		_, _ = io.Copy(io.Discard, conn)
	})

	_, err := network.Dial(context.Background(), address)
	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("Dial() error = %v, want greeting timeout", err)
	}
}

func TestDialGreetingCancellationReturnsContextErrorAfterRead(t *testing.T) {
	accepted := make(chan struct{})
	releaseGreeting := make(chan struct{})
	address := startTCPServer(t, func(conn net.Conn) {
		close(accepted)
		<-releaseGreeting
		writeTCPLine(conn, "OK hello proto=1")
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		connection, err := network.Dial(ctx, address)
		if connection != nil {
			_ = connection.Close()
		}
		result <- err
	}()

	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Fatal("server did not accept the connection")
	}
	cancel()
	close(releaseGreeting)

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Dial() error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Dial() did not return after the greeting read completed")
	}
}

func dialForAuthentication(t *testing.T, handler func(net.Conn)) *network.Connection {
	t.Helper()

	address := startTCPServer(t, func(conn net.Conn) {
		writeTCPLine(conn, "OK hello proto=1")
		handler(conn)
	})
	connection, err := network.Dial(context.Background(), address)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	t.Cleanup(func() {
		_ = connection.Close()
	})
	return connection
}

func TestConnectionAuthenticateSuccess(t *testing.T) {
	connection := dialForAuthentication(t, func(conn net.Conn) {
		line, err := readTCPLine(conn)
		if err != nil || line != "CONNECT alice" {
			t.Errorf("received %q, %v; want CONNECT alice", line, err)
			return
		}
		writeTCPLine(conn, "OK connected")
	})

	if err := connection.Authenticate("alice"); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if err := connection.Authenticate("alice"); !errors.Is(err, network.ErrConnectionAuthenticated) {
		t.Fatalf("second Authenticate() error = %v, want already authenticated", err)
	}
}

func TestConnectionNameInUseCanRetry(t *testing.T) {
	connection := dialForAuthentication(t, func(conn net.Conn) {
		first, err := readTCPLine(conn)
		if err != nil || first != "CONNECT alice" {
			t.Errorf("first request = %q, %v", first, err)
			return
		}
		writeTCPLine(conn, "ERR 201 NAME_IN_USE")

		second, err := readTCPLine(conn)
		if err != nil || second != "CONNECT bob" {
			t.Errorf("second request = %q, %v", second, err)
			return
		}
		writeTCPLine(conn, "OK connected")
	})

	if err := connection.Authenticate("alice"); !errors.Is(err, network.ErrNameInUse) {
		t.Fatalf("first Authenticate() error = %v, want ErrNameInUse", err)
	}
	if err := connection.Authenticate("bob"); err != nil {
		t.Fatalf("retry Authenticate() error = %v", err)
	}
}

func TestConnectionRejectsInvalidUsername(t *testing.T) {
	requestResult := make(chan error, 1)
	address := startTCPServer(t, func(conn net.Conn) {
		writeTCPLine(conn, "OK hello proto=1")
		_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		_, err := readTCPLine(conn)
		requestResult <- err
	})
	connection, err := network.Dial(context.Background(), address)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer connection.Close()

	if err := connection.Authenticate("bad name"); !errors.Is(err, protocol.ErrInvalidUsername) {
		t.Fatalf("Authenticate() error = %v, want ErrInvalidUsername", err)
	}
	select {
	case err := <-requestResult:
		var netErr net.Error
		if !errors.As(err, &netErr) || !netErr.Timeout() {
			t.Fatalf("server read error = %v, want no request before timeout", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not finish checking for an invalid username request")
	}
}

func TestConnectionConcurrentAuthentication(t *testing.T) {
	requestReceived := make(chan struct{})
	connection := dialForAuthentication(t, func(conn net.Conn) {
		line, err := readTCPLine(conn)
		if err != nil || line != "CONNECT alice" {
			t.Errorf("request = %q, %v; want CONNECT alice", line, err)
			return
		}
		close(requestReceived)
		writeTCPLine(conn, "OK connected")
	})

	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			results <- connection.Authenticate("alice")
		}()
	}
	select {
	case <-requestReceived:
	case <-time.After(time.Second):
		t.Fatal("authentication request was not sent")
	}

	var succeeded, alreadyAuthenticated int
	for i := 0; i < 2; i++ {
		err := <-results
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, network.ErrConnectionAuthenticated):
			alreadyAuthenticated++
		default:
			t.Errorf("Authenticate() error = %v", err)
		}
	}
	if succeeded != 1 || alreadyAuthenticated != 1 {
		t.Fatalf("successes = %d, already-authenticated = %d; want 1 each", succeeded, alreadyAuthenticated)
	}
}

func TestConnectionUnexpectedAuthenticationResponse(t *testing.T) {
	connection := dialForAuthentication(t, func(conn net.Conn) {
		_, _ = readTCPLine(conn)
		writeTCPLine(conn, "READY")
	})

	if err := connection.Authenticate("alice"); err == nil ||
		!strings.Contains(err.Error(), "unexpected authentication response") {
		t.Fatalf("Authenticate() error = %v, want unexpected response", err)
	}
}

func TestConnectionAuthenticationServerError(t *testing.T) {
	connection := dialForAuthentication(t, func(conn net.Conn) {
		_, _ = readTCPLine(conn)
		writeTCPLine(conn, "ERR 500 INTERNAL")
	})

	if err := connection.Authenticate("alice"); !errors.Is(err, network.ErrAuthentication) {
		t.Fatalf("Authenticate() error = %v, want ErrAuthentication", err)
	}
}

func TestConnectionCloseDuringAuthentication(t *testing.T) {
	connection := dialForAuthentication(t, func(conn net.Conn) {
		_, _ = readTCPLine(conn)
		_ = conn.Close()
	})

	if err := connection.Authenticate("alice"); err == nil {
		t.Fatal("Authenticate() succeeded after the server closed the connection")
	}
}
