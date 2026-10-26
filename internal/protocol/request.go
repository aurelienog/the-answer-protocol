package protocol

import (
	"fmt"
	"strings"
)

// Request representa una línea de entrada enviada por un cliente TAP.
type Request struct {
	Command string
	Args    []string
	RawArgs string
}

// ParseRequest interpreta una línea TAP enviada por el cliente.
//
// Los nombres de los comandos no distinguen mayúsculas de minúsculas.
// RawArgs conserva el texto original de los argumentos, algo útil
// para mensajes CHAT.
func ParseRequest(line string) (Request, error) {
	line = strings.TrimSpace(line)

	if line == "" {
		return Request{}, fmt.Errorf("empty TAP request")
	}

	commandEnd := strings.IndexAny(line, " \t")

	if commandEnd == -1 {
		return Request{
			Command: strings.ToUpper(line),
		}, nil
	}

	command := line[:commandEnd]
	rawArgs := strings.TrimSpace(line[commandEnd:])

	var args []string
	if rawArgs != "" {
		args = strings.Fields(rawArgs)
	}

	return Request{
		Command: strings.ToUpper(command),
		Args:    args,
		RawArgs: rawArgs,
	}, nil
}
