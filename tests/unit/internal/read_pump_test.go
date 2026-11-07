package internal_test

import (
	"net"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"the-answer-protocol/internal/server"
)

const testTimeout = 2 * time.Second

// runReadPump executes ReadPump with a simulated connection and returns
// all events produced before the reader finishes.
func runReadPump(t *testing.T, chunks ...string) []server.ReadEvent {
	t.Helper()

	serverConnection, clientConnection := net.Pipe()
	events := make(chan server.ReadEvent, 16)
	stop := make(chan struct{})

	go server.ReadPump(serverConnection, events, stop)

	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		defer clientConnection.Close()

		for _, chunk := range chunks {
			if _, err := clientConnection.Write([]byte(chunk)); err != nil {
				return
			}
		}
	}()

	var receivedEvents []server.ReadEvent

	for {
		select {
		case event, open := <-events:
			if !open {
				select {
				case <-writeDone:
				case <-time.After(testTimeout):
					t.Fatal("timed out waiting for the client writer to finish")
				}

				_ = serverConnection.Close()
				return receivedEvents
			}

			receivedEvents = append(receivedEvents, event)

		case <-time.After(testTimeout):
			close(stop)
			_ = clientConnection.Close()
			_ = serverConnection.Close()
			t.Fatal("timed out waiting for ReadPump events")
			return nil
		}
	}
}

func TestReadPump_ReceivesSingleLine(t *testing.T) {
	events := runReadPump(t, "LOOK\n")

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Err != nil {
		t.Fatalf("expected no error, got %v", events[0].Err)
	}

	if events[0].Line != "LOOK" {
		t.Errorf("expected line %q, got %q", "LOOK", events[0].Line)
	}
}

func TestReadPump_ReceivesMultipleLines(t *testing.T) {
	events := runReadPump(t, "LOOK\nMOVE north\nQUIT\n")

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	expectedLines := []string{"LOOK", "MOVE north", "QUIT"}

	for index, expectedLine := range expectedLines {
		if events[index].Err != nil {
			t.Errorf("event %d: unexpected error: %v", index, events[index].Err)
			continue
		}

		if events[index].Line != expectedLine {
			t.Errorf(
				"event %d: expected line %q, got %q",
				index,
				expectedLine,
				events[index].Line,
			)
		}
	}
}

func TestReadPump_HandlesFragmentedLine(t *testing.T) {
	events := runReadPump(t, "MO", "VE ", "north", "\n")

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Err != nil {
		t.Fatalf("expected no error, got %v", events[0].Err)
	}

	if events[0].Line != "MOVE north" {
		t.Errorf("expected line %q, got %q", "MOVE north", events[0].Line)
	}
}

func TestReadPump_HandlesUTF8SequenceFragmentedAcrossReads(t *testing.T) {
	line := "CHAT ¡Hola, 世界! 🌍"
	chunks := make([]string, 0, len(line)+1)
	for i := 0; i < len(line); i++ {
		chunks = append(chunks, line[i:i+1])
	}
	chunks = append(chunks, "\n")

	events := runReadPump(t, chunks...)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Err != nil {
		t.Fatalf("expected fragmented UTF-8 to be accepted, got %v", events[0].Err)
	}
	if events[0].Line != line {
		t.Errorf("expected line %q, got %q", line, events[0].Line)
	}
}

func TestReadPump_AcceptsMaximumLineLength(t *testing.T) {
	line := strings.Repeat("a", serverMaxLineBytesForTest)

	events := runReadPump(t, line+"\n")

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Err != nil {
		t.Fatalf("expected maximum-length line to be accepted, got %v", events[0].Err)
	}

	if len(events[0].Line) != serverMaxLineBytesForTest {
		t.Errorf(
			"expected line length %d, got %d",
			serverMaxLineBytesForTest,
			len(events[0].Line),
		)
	}
}

func TestReadPump_AcceptsMaximumLineLengthWithCRLF(t *testing.T) {
	line := strings.Repeat("a", serverMaxLineBytesForTest)

	events := runReadPump(t, line+"\r\n")

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Err != nil {
		t.Fatalf("expected maximum-length CRLF line to be accepted, got %v", events[0].Err)
	}

	if events[0].Line != line {
		t.Errorf("expected CRLF to be removed from the line")
	}
}

func TestReadPumpMeasuresMaximumLengthInBytes(t *testing.T) {
	maxLengthLine := strings.Repeat("a", serverMaxLineBytesForTest-2) + "é"
	events := runReadPump(t, maxLengthLine+"\n")

	if len(events) != 1 || events[0].Err != nil {
		t.Fatalf("expected %d-byte UTF-8 line to be accepted, got %+v", serverMaxLineBytesForTest, events)
	}
	if got := len([]byte(events[0].Line)); got != serverMaxLineBytesForTest {
		t.Fatalf("line byte length = %d, want %d", got, serverMaxLineBytesForTest)
	}

	overLimitLine := strings.Repeat("a", serverMaxLineBytesForTest-1) + "é"
	events = runReadPump(t, overLimitLine+"\n")

	if len(events) != 1 || events[0].ProtocolError == nil {
		t.Fatalf("expected %d-byte UTF-8 line to be rejected, got %+v", serverMaxLineBytesForTest+1, events)
	}
	if !strings.Contains(events[0].ProtocolError.Error(), "exceeds maximum length") {
		t.Errorf("expected maximum-length error, got %v", events[0].ProtocolError)
	}
}

func TestReadPump_RejectsLineExceedingMaximumLength(t *testing.T) {
	line := strings.Repeat("a", serverMaxLineBytesForTest+1)

	events := runReadPump(t, line+"\n")

	if len(events) != 1 {
		t.Fatalf("expected 1 error event, got %d events", len(events))
	}

	if events[0].ProtocolError == nil {
		t.Fatal("expected an error for a line exceeding the maximum length")
	}
	if !strings.Contains(events[0].ProtocolError.Error(), "exceeds maximum length") {
		t.Errorf("expected maximum-length error, got %v", events[0].ProtocolError)
	}
}

func TestReadPump_RejectsInvalidUTF8(t *testing.T) {
	invalidUTF8 := string([]byte{'L', 'O', 'O', 'K', ' ', 0xff, '\n'})

	if utf8.ValidString(strings.TrimSuffix(invalidUTF8, "\n")) {
		t.Fatal("test input should contain invalid UTF-8")
	}

	events := runReadPump(t, invalidUTF8)

	if len(events) != 1 {
		t.Fatalf("expected 1 error event, got %d events", len(events))
	}

	if events[0].ProtocolError == nil {
		t.Fatal("expected an error for invalid UTF-8")
	}
	if events[0].ProtocolError.Error() != "message is not valid UTF-8" {
		t.Errorf("expected invalid UTF-8 error, got %v", events[0].ProtocolError)
	}
}

func TestReadPump_RejectsControlCharacters(t *testing.T) {
	events := runReadPump(t, "LOOK\x00\n")

	if len(events) != 1 {
		t.Fatalf("expected 1 error event, got %d events", len(events))
	}

	if events[0].ProtocolError == nil {
		t.Fatal("expected an error for a control character")
	}
}

func TestReadPump_AcceptsValidUTF8(t *testing.T) {
	line := "CHAT ¡Hola, mundo! 你好 🌍"

	events := runReadPump(t, line+"\n")

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Err != nil {
		t.Fatalf("expected valid UTF-8 to be accepted, got %v", events[0].Err)
	}

	if events[0].Line != line {
		t.Errorf("expected line %q, got %q", line, events[0].Line)
	}
}

// Keep this value aligned with the server's documented protocol limit.
// The constant is duplicated here because maxLineBytes is unexported.
const serverMaxLineBytesForTest = 1024

func TestReadPump_RejectsUnterminatedLineAtEOF(t *testing.T) {
	events := runReadPump(t, "LOOK")

	if len(events) != 1 {
		t.Fatalf("expected 1 error event, got %d events: %+v", len(events), events)
	}

	if events[0].ProtocolError == nil {
		t.Fatal("expected an error for a message without LF")
	}
	if !strings.Contains(events[0].ProtocolError.Error(), "unterminated message") {
		t.Errorf("expected unterminated-message error, got %v", events[0].ProtocolError)
	}
}

func TestReadPumpContinuesAfterInvalidLine(t *testing.T) {
	events := runReadPump(t, "LOOK\n", "BAD\x00\n", "MOVE north\n")

	if len(events) != 3 {
		t.Fatalf("expected valid, malformed, and valid events, got %+v", events)
	}
	if events[0].Err != nil || events[0].Line != "LOOK" {
		t.Errorf("first event = %+v, want LOOK without error", events[0])
	}
	if events[1].ProtocolError == nil || !strings.Contains(events[1].ProtocolError.Error(), "control characters") {
		t.Errorf("second event = %+v, want control-character error", events[1])
	}
	if events[2].Err != nil || events[2].Line != "MOVE north" {
		t.Errorf("third event = %+v, want MOVE north after malformed line", events[2])
	}
}
