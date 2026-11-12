package network

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	MaxLineBytes     = 1024
	WriteTimeout     = 10 * time.Second
	readerBufferSize = 4096
)

var (
	ErrLineTooLong = errors.New(
		"TAP message exceeds maximum length",
	)

	ErrInvalidUTF8 = errors.New(
		"TAP message is not valid UTF-8",
	)

	ErrControlChar = errors.New(
		"TAP message contains control characters",
	)

	ErrInvalidLineEnding = errors.New(
		"TAP message contains invalid line endings",
	)

	ErrUnterminatedLine = errors.New(
		"TAP message is not terminated with LF",
	)
)

// Transport handles line-oriented TCP communication.
//
// It provides synchronized reads and writes.
// Message interpretation belongs to the protocol package.
type Transport struct {
	conn net.Conn

	reader *bufio.Reader

	readMu  sync.Mutex
	writeMu sync.Mutex

	closeOnce sync.Once
	closeErr  error

	writeTimeout time.Duration
}

// NewTransport creates a transport over an existing TCP connection.
func NewTransport(conn net.Conn) *Transport {
	t := &Transport{
		conn:         conn,
		writeTimeout: WriteTimeout,
	}

	if conn != nil {
		t.reader = bufio.NewReaderSize(conn, readerBufferSize)
	}

	return t
}

// ReadLine reads one complete TAP message.
//
// LF and CRLF are accepted.
// Oversized messages are discarded until LF.
// Only one logical reader should use this method.
func (t *Transport) ReadLine() (string, error) {
	if t == nil || t.conn == nil || t.reader == nil {
		return "", net.ErrClosed
	}

	t.readMu.Lock()
	defer t.readMu.Unlock()

	line, err := t.readBoundedLine(MaxLineBytes)
	if err != nil {
		return "", err
	}

	// Accept CRLF.
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}

	if len(line) > MaxLineBytes {
		return "", ErrLineTooLong
	}

	message := string(line)

	if !utf8.ValidString(message) {
		return "", ErrInvalidUTF8
	}

	if strings.IndexFunc(message, unicode.IsControl) >= 0 {
		return "", ErrControlChar
	}

	return message, nil
}

// readBoundedLine reads until LF while limiting memory usage.
//
// It allows MaxLineBytes+1 bytes to accommodate a possible CR
// immediately before LF.
func (t *Transport) readBoundedLine(maxBytes int) ([]byte, error) {
	line := make([]byte, 0, maxBytes+1)
	tooLong := false

	for {
		chunk, err := t.reader.ReadSlice('\n')

		// Remove LF from the received chunk.
		if len(chunk) > 0 && chunk[len(chunk)-1] == '\n' {
			chunk = chunk[:len(chunk)-1]
		}

		// Accumulate only while within the allowed memory limit.
		if !tooLong {
			if len(line)+len(chunk) > maxBytes+1 {
				tooLong = true
				line = nil
			} else {
				line = append(line, chunk...)
			}
		}

		if err == nil {
			if tooLong {
				return nil, ErrLineTooLong
			}

			return line, nil
		}

		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}

		if errors.Is(err, io.EOF) {
			if tooLong {
				return nil, ErrLineTooLong
			}

			if len(line) == 0 {
				return nil, io.EOF
			}

			// A line exceeding the limit without a terminator
			// is still an oversized message, unless the extra
			// byte could be a CR awaiting LF.
			if len(line) > maxBytes &&
				!(len(line) == maxBytes+1 &&
					line[len(line)-1] == '\r') {
				return nil, ErrLineTooLong
			}

			return nil, ErrUnterminatedLine
		}

		return nil, fmt.Errorf("read TCP line: %w", err)
	}
}

// WriteLine sends one TAP message followed by LF.
//
// Writes are serialized to prevent concurrent TCP writes.
func (t *Transport) WriteLine(message string) error {
	if t == nil || t.conn == nil {
		return net.ErrClosed
	}

	if err := validateOutgoingMessage(message); err != nil {
		return err
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	if err := t.conn.SetWriteDeadline(
		time.Now().Add(t.writeTimeout),
	); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}

	data := append([]byte(message), '\n')

	_, writeErr := writeFull(t.conn, data)

	resetErr := t.conn.SetWriteDeadline(time.Time{})

	if writeErr != nil {
		return fmt.Errorf("write TCP message: %w", writeErr)
	}

	if resetErr != nil {
		return fmt.Errorf("reset write deadline: %w", resetErr)
	}

	return nil
}

// validateOutgoingMessage validates an outgoing TAP message.
func validateOutgoingMessage(message string) error {
	if len(message) > MaxLineBytes {
		return ErrLineTooLong
	}

	if !utf8.ValidString(message) {
		return ErrInvalidUTF8
	}

	if strings.ContainsAny(message, "\r\n") {
		return ErrInvalidLineEnding
	}

	if strings.IndexFunc(message, unicode.IsControl) >= 0 {
		return ErrControlChar
	}

	return nil
}

// writeFull ensures that the entire message is written.
func writeFull(writer io.Writer, data []byte) (int, error) {
	total := 0

	for total < len(data) {
		n, err := writer.Write(data[total:])
		total += n

		if err != nil {
			return total, err
		}

		if n == 0 {
			return total, io.ErrShortWrite
		}
	}

	return total, nil
}

// Close closes the underlying TCP connection.
//
// It is safe to call multiple times and concurrently.
func (t *Transport) Close() error {
	if t == nil || t.conn == nil {
		return nil
	}

	t.closeOnce.Do(func() {
		t.closeErr = t.conn.Close()
	})

	return t.closeErr
}

// SetReadDeadline sets the deadline for future read operations.
func (t *Transport) SetReadDeadline(deadline time.Time) error {
	if t == nil || t.conn == nil {
		return net.ErrClosed
	}

	if err := t.conn.SetReadDeadline(deadline); err != nil {
		return fmt.Errorf("set TCP read deadline: %w", err)
	}

	return nil
}

// | Test | Resultado esperado |
// |---|---|
// | Lectura con LF | Correcta |
// | Lectura con CRLF | Correcta |
// | Mensaje de 1024 bytes | Correcta |
// | Mensaje de 1024 bytes + CRLF | Correcta |
// | Mensaje superior a 1024 bytes | `ErrLineTooLong` |
// | Línea demasiado larga seguida de otra válida | La segunda se lee correctamente |
// | UTF-8 inválido | `ErrInvalidUTF8` |
// | Caracteres de control | `ErrControlChar` |
// | Conexión cerrada | Error de lectura/escritura |
// | Escrituras concurrentes | Mensajes completos, sin mezclarse |
// | Cierre durante una lectura bloqueada | La lectura termina |
// También necesitamos tests específicos para:
// - Greeting correcto.
// - Versión de protocolo incompatible.
// - Greeting incorrecto.
// - Timeout del greeting.
// - Autenticación correcta.
// - Nombre ocupado y segundo intento.
// - Username inválido.
// - Dos autenticaciones concurrentes.
// - Cancelación del contexto durante el greeting.