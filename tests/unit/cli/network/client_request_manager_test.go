package network_test

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"the-answer-protocol/frontend/cli/network"
	"the-answer-protocol/internal/protocol"
)

func TestClientSendRequiresAuthentication(t *testing.T) {
	connection := dialForAuthentication(t, func(net.Conn) {})
	client, err := network.NewClient(context.Background(), connection)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	if _, err := client.Send(context.Background(), "LOOK"); !errors.Is(err, network.ErrClientNotAuthenticated) {
		t.Fatalf("Send() error = %v, want ErrClientNotAuthenticated", err)
	}
}

func TestClientAuthenticateSendAndReceiveEvent(t *testing.T) {
	connection := dialForAuthentication(t, func(conn net.Conn) {
		if line, err := readTCPLine(conn); err != nil || line != "CONNECT alice" {
			t.Errorf("authentication request = %q, %v", line, err)
			return
		}
		writeTCPLine(conn, "OK connected")
		if line, err := readTCPLine(conn); err != nil || line != "LOOK" {
			t.Errorf("command = %q, %v", line, err)
			return
		}
		writeTCPLine(conn, "OK room=harbor")
		writeTCPLine(conn, "EVT CHAT alice: hello")
	})

	client, err := network.NewClient(context.Background(), connection)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()
	if err := client.Authenticate("alice"); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	response, err := client.Send(context.Background(), "LOOK")
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if response.Kind != protocol.KindOK || response.Payload != "room=harbor" {
		t.Fatalf("response = %#v", response)
	}

	select {
	case event := <-client.Events():
		if event.Message.Kind != protocol.KindEVT ||
			event.Message.EventType != "CHAT" ||
			event.Message.Data != "alice: hello" {
			t.Fatalf("event = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for asynchronous event")
	}
}

func TestRequestManagerAllowsOnlyOnePendingRequest(t *testing.T) {
	responses := make(chan network.IncomingMessage, 2)
	manager, err := network.NewRequestManager(context.Background(), responses)
	if err != nil {
		t.Fatalf("NewRequestManager() error = %v", err)
	}
	defer manager.Close()

	firstSent := make(chan struct{})
	secondSent := make(chan struct{})
	results := make(chan protocol.Message, 2)
	errs := make(chan error, 2)

	go func() {
		message, err := manager.Do(context.Background(), func(context.Context) error {
			close(firstSent)
			return nil
		})
		results <- message
		errs <- err
	}()
	select {
	case <-firstSent:
	case <-time.After(time.Second):
		t.Fatal("first request was not sent")
	}

	go func() {
		message, err := manager.Do(context.Background(), func(context.Context) error {
			close(secondSent)
			return nil
		})
		results <- message
		errs <- err
	}()

	select {
	case <-secondSent:
		t.Fatal("second request was sent while the first was pending")
	case <-time.After(30 * time.Millisecond):
	}

	responses <- network.IncomingMessage{
		Message: protocol.OK("first"),
		Raw:     "OK first",
	}
	select {
	case <-secondSent:
	case <-time.After(time.Second):
		t.Fatal("second request was not sent after the first completed")
	}
	responses <- network.IncomingMessage{
		Message: protocol.OK("second"),
		Raw:     "OK second",
	}

	for i, want := range []string{"first", "second"} {
		if err := <-errs; err != nil {
			t.Fatalf("request %d error = %v", i+1, err)
		}
		if got := (<-results).Payload; got != want {
			t.Fatalf("request %d payload = %q, want %q", i+1, got, want)
		}
	}
}

func TestRequestManagerHandlesOKAndERRResponses(t *testing.T) {
	tests := []struct {
		name     string
		message  protocol.Message
		wantText string
		wantErr  bool
		wantCode int
	}{
		{
			name:     "OK",
			message:  protocol.OK("done"),
			wantText: "done",
		},
		{
			name:     "ERR",
			message:  protocol.ERR(404, "NOT_FOUND"),
			wantErr:  true,
			wantCode: 404,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responses := make(chan network.IncomingMessage, 1)
			responses <- network.IncomingMessage{Message: tt.message}
			manager, err := network.NewRequestManager(context.Background(), responses)
			if err != nil {
				t.Fatalf("NewRequestManager() error = %v", err)
			}
			defer manager.Close()

			got, err := manager.Do(context.Background(), func(context.Context) error {
				return nil
			})
			if tt.wantErr {
				var serverErr *protocol.ServerError
				if !errors.As(err, &serverErr) || serverErr.Code != tt.wantCode {
					t.Fatalf("Do() error = %v, want server error %d", err, tt.wantCode)
				}
				return
			}
			if err != nil {
				t.Fatalf("Do() error = %v", err)
			}
			if got.Payload != tt.wantText {
				t.Fatalf("Do() payload = %q, want %q", got.Payload, tt.wantText)
			}
		})
	}
}

func TestRequestManagerCloseInterruptsPendingRequest(t *testing.T) {
	manager, err := network.NewRequestManager(
		context.Background(),
		make(chan network.IncomingMessage),
	)
	if err != nil {
		t.Fatalf("NewRequestManager() error = %v", err)
	}

	sent := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		_, err := manager.Do(context.Background(), func(context.Context) error {
			close(sent)
			return nil
		})
		result <- err
	}()

	select {
	case <-sent:
	case <-time.After(time.Second):
		t.Fatal("request was not sent")
	}
	manager.Close()
	select {
	case err := <-result:
		if !errors.Is(err, network.ErrRequestManagerClosed) {
			t.Fatalf("Do() error = %v, want ErrRequestManagerClosed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("pending request did not stop after Close")
	}
}

func TestClientCloseInterruptsPendingRequest(t *testing.T) {
	commandReceived := make(chan struct{})
	connection := dialForAuthentication(t, func(conn net.Conn) {
		_, _ = readTCPLine(conn)
		writeTCPLine(conn, "OK connected")
		if _, err := readTCPLine(conn); err != nil {
			return
		}
		close(commandReceived)
		_, _ = io.Copy(io.Discard, conn)
	})
	client, err := network.NewClient(context.Background(), connection)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if err := client.Authenticate("alice"); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	result := make(chan error, 1)
	go func() {
		_, err := client.Send(context.Background(), "LOOK")
		result <- err
	}()
	select {
	case <-commandReceived:
	case <-time.After(time.Second):
		t.Fatal("server did not receive request")
	}
	if err := client.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	select {
	case err := <-result:
		if !errors.Is(err, network.ErrRequestManagerClosed) {
			t.Fatalf("Send() error = %v, want ErrRequestManagerClosed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("pending Send() did not return after Client.Close")
	}
}

func TestClientClosesAfterFatalReaderError(t *testing.T) {
	connection := dialForAuthentication(t, func(conn net.Conn) {
		_, _ = readTCPLine(conn)
		writeTCPLine(conn, "OK connected")
		_ = conn.Close()
	})
	client, err := network.NewClient(context.Background(), connection)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if err := client.Authenticate("alice"); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	select {
	case <-client.Done():
	case <-time.After(time.Second):
		t.Fatal("client did not close after the reader encountered EOF")
	}
}

func TestWriterReportsFatalTransportError(t *testing.T) {
	peer, conn := net.Pipe()
	_ = peer.Close()
	transport := network.NewTransport(conn)
	defer transport.Close()

	outbox := network.NewOutbox()
	defer outbox.Close()
	fatalErrors := make(chan error, 1)
	writer, err := network.NewWriter(transport, outbox, func(err error) {
		fatalErrors <- err
	})
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	writer.Start(ctx)

	if err := writer.Send(ctx, "LOOK"); err == nil {
		t.Fatal("Send() succeeded with a closed transport")
	}
	select {
	case err := <-fatalErrors:
		if err == nil {
			t.Fatal("writer reported a nil fatal error")
		}
	case <-time.After(time.Second):
		t.Fatal("writer did not report its fatal transport error")
	}
}
