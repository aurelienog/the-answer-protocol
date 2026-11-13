package network_test

import (
	"bufio"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"the-answer-protocol/frontend/cli/network"
)

func TestTransportReadLine(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    string
		wantErr error
	}{
		{name: "LF", input: []byte("hello\n"), want: "hello"},
		{name: "CRLF", input: []byte("hello\r\n"), want: "hello"},
		{
			name: "exactly maximum bytes",
			input: append(
				[]byte(strings.Repeat("a", network.MaxIncomingLineBytes)),
				'\n',
			),
			want: strings.Repeat("a", network.MaxIncomingLineBytes),
		},
		{
			name: "maximum bytes followed by CRLF",
			input: append(
				[]byte(strings.Repeat("a", network.MaxIncomingLineBytes)+"\r"),
				'\n',
			),
			want: strings.Repeat("a", network.MaxIncomingLineBytes),
		},
		{
			name:    "message exceeds maximum",
			input:   []byte(strings.Repeat("a", network.MaxIncomingLineBytes+1) + "\n"),
			wantErr: network.ErrLineTooLong,
		},
		{
			name:    "invalid UTF-8",
			input:   []byte{0xff, '\n'},
			wantErr: network.ErrInvalidUTF8,
		},
		{
			name:    "control character",
			input:   []byte("hello\tworld\n"),
			wantErr: network.ErrControlChar,
		},
		{
			name:    "missing LF",
			input:   []byte("hello"),
			wantErr: network.ErrUnterminatedLine,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peer, conn := net.Pipe()
			transport := network.NewTransport(conn)
			t.Cleanup(func() {
				_ = transport.Close()
				_ = peer.Close()
			})

			go func() {
				_, _ = peer.Write(tt.input)
				_ = peer.Close()
			}()

			got, err := transport.ReadLine()
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ReadLine() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadLine() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ReadLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTransportReadLineDiscardsOversizedLine(t *testing.T) {
	peer, conn := net.Pipe()
	transport := network.NewTransport(conn)
	t.Cleanup(func() {
		_ = transport.Close()
		_ = peer.Close()
	})

	go func() {
		_, _ = io.WriteString(
			peer,
			strings.Repeat("a", network.MaxIncomingLineBytes+20)+"\nvalid\n",
		)
	}()

	if _, err := transport.ReadLine(); !errors.Is(err, network.ErrLineTooLong) {
		t.Fatalf("first ReadLine() error = %v, want ErrLineTooLong", err)
	}
	if got, err := transport.ReadLine(); err != nil || got != "valid" {
		t.Fatalf("second ReadLine() = %q, %v; want valid line", got, err)
	}
}

func TestTransportReadLineUnblocksOnClose(t *testing.T) {
	peer, conn := net.Pipe()
	transport := network.NewTransport(conn)
	defer peer.Close()

	done := make(chan error, 1)
	go func() {
		_, err := transport.ReadLine()
		done <- err
	}()

	if err := transport.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("ReadLine() succeeded after connection close")
		}
	case <-time.After(time.Second):
		t.Fatal("ReadLine() remained blocked after Close")
	}
}

func TestTransportConcurrentWritesAreSerialized(t *testing.T) {
	const count = 32

	peer, conn := net.Pipe()
	transport := network.NewTransport(conn)
	t.Cleanup(func() {
		_ = transport.Close()
		_ = peer.Close()
	})

	got := make(chan string, count)
	go func() {
		reader := bufio.NewReader(peer)
		for i := 0; i < count; i++ {
			line, err := reader.ReadString('\n')
			if err != nil {
				got <- "read error: " + err.Error()
				return
			}
			got <- strings.TrimSuffix(line, "\n")
		}
	}()

	var writers sync.WaitGroup
	writeErrors := make(chan error, count)
	for i := 0; i < count; i++ {
		writers.Add(1)
		go func(i int) {
			defer writers.Done()
			if err := transport.WriteLine(strings.Repeat(string(rune('a'+i%26)), i+1)); err != nil {
				writeErrors <- err
			}
		}(i)
	}
	writers.Wait()
	close(writeErrors)
	for err := range writeErrors {
		t.Errorf("WriteLine() error = %v", err)
	}

	want := make(map[string]int, count)
	for i := 0; i < count; i++ {
		want[strings.Repeat(string(rune('a'+i%26)), i+1)]++
	}
	for i := 0; i < count; i++ {
		select {
		case line := <-got:
			if want[line] == 0 {
				t.Errorf("received incomplete or unexpected line %q", line)
				continue
			}
			want[line]--
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for concurrent writes")
		}
	}
}

func TestTransportCloseIsConcurrentAndIdempotent(t *testing.T) {
	peer, conn := net.Pipe()
	transport := network.NewTransport(conn)
	defer peer.Close()

	const count = 32
	errs := make(chan error, count)
	var callers sync.WaitGroup
	for i := 0; i < count; i++ {
		callers.Add(1)
		go func() {
			defer callers.Done()
			errs <- transport.Close()
		}()
	}
	callers.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}
	if err := transport.Close(); err != nil {
		t.Errorf("repeated Close() error = %v", err)
	}
}

func TestTransportWriteAfterClose(t *testing.T) {
	peer, conn := net.Pipe()
	transport := network.NewTransport(conn)
	defer peer.Close()

	if err := transport.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := transport.WriteLine("message"); err == nil {
		t.Fatal("WriteLine() succeeded after Close")
	}
}

type shortWriteDeadlineConn struct {
	net.Conn
	timeout time.Duration
}

func (c shortWriteDeadlineConn) SetWriteDeadline(time.Time) error {
	return c.Conn.SetWriteDeadline(time.Now().Add(c.timeout))
}

func TestTransportWriteTimeout(t *testing.T) {
	peer, conn := net.Pipe()
	transport := network.NewTransport(shortWriteDeadlineConn{
		Conn:    conn,
		timeout: 20 * time.Millisecond,
	})
	t.Cleanup(func() {
		_ = transport.Close()
		_ = peer.Close()
	})

	err := transport.WriteLine("blocked")
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("WriteLine() error = %v, want deadline exceeded", err)
	}
}
