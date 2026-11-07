package server_test

import (
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"the-answer-protocol/internal/protocol"
	"the-answer-protocol/internal/server"
)

func TestWritePumpDeliversResponsesAndEventsInOrder(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	session := &server.Session{
		ID:         "session-1",
		PlayerName: "Alice",
		Connection: serverConn,
		Outbox:     make(chan string, 2),
	}
	done := make(chan struct{})
	go server.WritePump(session, done)

	if err := server.SendMessage(session, protocol.OK("connected")); err != nil {
		t.Fatalf("SendMessage(response) returned an error: %v", err)
	}
	if err := server.SendMessage(session, protocol.EVT("CHAT", "Alice: hello")); err != nil {
		t.Fatalf("SendMessage(event) returned an error: %v", err)
	}
	session.CloseOutbox()

	want := "OK connected\nEVT CHAT Alice: hello\n"
	got := make([]byte, len(want))
	if _, err := io.ReadFull(clientConn, got); err != nil {
		t.Fatalf("read messages from WritePump: %v", err)
	}
	if string(got) != want {
		t.Fatalf("messages = %q, want %q", got, want)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("WritePump did not finish after the outbox was closed")
	}
}

func TestWritePumpClosesConnectionAfterWriteFailure(t *testing.T) {
	writeErr := errors.New("connection write failed")
	conn := &failingWriteConn{writeErr: writeErr}
	session := &server.Session{
		ID:         "session-1",
		PlayerName: "Alice",
		Connection: conn,
		Outbox:     make(chan string, 1),
	}
	done := make(chan struct{})

	if err := session.Enqueue("message\n"); err != nil {
		t.Fatalf("Enqueue() returned an error: %v", err)
	}

	go server.WritePump(session, done)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("WritePump did not stop after the write failed")
	}

	if got := conn.closeCount(); got != 1 {
		t.Errorf("connection Close() calls = %d, want 1", got)
	}
}

func TestSessionEnqueueReportsFullOutbox(t *testing.T) {
	session := &server.Session{
		Outbox: make(chan string, 1),
	}

	if err := session.Enqueue("first"); err != nil {
		t.Fatalf("Enqueue(first) returned an error: %v", err)
	}

	if err := session.Enqueue("second"); !errors.Is(err, server.ErrOutboxFull) {
		t.Fatalf("Enqueue(second) error = %v, want %v", err, server.ErrOutboxFull)
	}
}

func TestRegistryQueueReportsFullOutbox(t *testing.T) {
	registry := server.NewRegistry()
	session := newRegistryTestSession("session-1", "Alice")
	if err := registry.Add(session); err != nil {
		t.Fatalf("Add(session) returned an error: %v", err)
	}

	for i := 0; i < cap(session.Outbox); i++ {
		if err := session.Enqueue("pending"); err != nil {
			t.Fatalf("fill outbox at message %d: %v", i, err)
		}
	}

	err := registry.Queue(session.ID, protocol.OK("response"))
	if !errors.Is(err, server.ErrOutboxFull) {
		t.Fatalf("Queue() error = %v, want %v", err, server.ErrOutboxFull)
	}
}

func TestRegistryBroadcastSkipsFullOutboxWithoutBlockingOtherSessions(t *testing.T) {
	registry := server.NewRegistry()
	fullSession := newRegistryTestSession("session-full", "Alice")
	availableSession := newRegistryTestSession("session-available", "Bob")

	for _, session := range []*server.Session{fullSession, availableSession} {
		if err := registry.Add(session); err != nil {
			t.Fatalf("Add(%s) returned an error: %v", session.ID, err)
		}
	}

	for i := 0; i < cap(fullSession.Outbox); i++ {
		if err := fullSession.Enqueue("pending"); err != nil {
			t.Fatalf("fill full session outbox at message %d: %v", i, err)
		}
	}

	registry.Broadcast(protocol.EVT("ROOM", "PRESENCE ENTER Bob"))

	select {
	case got := <-availableSession.Outbox:
		if got != "EVT ROOM PRESENCE ENTER Bob\n" {
			t.Fatalf("broadcast message = %q", got)
		}
	default:
		t.Fatal("broadcast was not delivered to the session with available capacity")
	}

	if got := len(fullSession.Outbox); got != cap(fullSession.Outbox) {
		t.Errorf("full outbox length = %d, want %d", got, cap(fullSession.Outbox))
	}
}

type failingWriteConn struct {
	net.Conn
	writeErr error
	mu       sync.Mutex
	closes   int
}

func (c *failingWriteConn) SetWriteDeadline(time.Time) error {
	return nil
}

func (c *failingWriteConn) Write([]byte) (int, error) {
	return 0, c.writeErr
}

func (c *failingWriteConn) Close() error {
	c.mu.Lock()
	c.closes++
	c.mu.Unlock()
	return nil
}

func (c *failingWriteConn) closeCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closes
}
