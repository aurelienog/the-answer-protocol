package protocol

import (
	"fmt"
	"strconv"
	"strings"
)

// ServerError represents an error returned by the TAP server.
type ServerError struct {
	Code int
	Name string
}

func (e *ServerError) Error() string {
	if e == nil {
		return "unknown TAP server error"
	}

	return fmt.Sprintf("TAP server error %03d: %s", e.Code, e.Name)
}

func (e *ServerError) Is(target error) bool {
	other, ok := target.(*ServerError)
	if !ok || e == nil || other == nil {
		return false
	}

	return e.Code == other.Code
}

// ParseServerError parses an ERR message payload.
func ParseServerError(payload string) (*ServerError, error) {
	codeText, name, found := strings.Cut(payload, " ")

	if !found || name == "" {
		return nil, fmt.Errorf("malformed TAP server error: %q", payload)
	}

	if len(codeText) != 3 {
		return nil, fmt.Errorf("invalid TAP error code: %q", codeText)
	}

	code, err := strconv.Atoi(codeText)
	if err != nil || code < 100 || code > 999 {
		return nil, fmt.Errorf("invalid TAP error code: %q", codeText)
	}

	if strings.ContainsAny(name, " \t\r\n") {
		return nil, fmt.Errorf("invalid TAP error name: %q", name)
	}

	return &ServerError{
		Code: code,
		Name: name,
	}, nil
}