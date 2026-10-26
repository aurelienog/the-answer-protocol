package internal_test

import (
	"testing"

	"the-answer-protocol/internal/protocol"
)

func TestParseRequestRejectsEmptyInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "spaces only", input: "   "},
		{name: "tabs only", input: "\t\t"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := protocol.ParseRequest(test.input)
			if err == nil {
				t.Fatal("expected an error for empty input")
			}
		})
	}
}

func TestParseRequestNormalizesCommand(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "look", want: "LOOK"},
		{input: "Look", want: "LOOK"},
		{input: "LOOK", want: "LOOK"},
		{input: "  move north  ", want: "MOVE"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			request, err := protocol.ParseRequest(test.input)
			if err != nil {
				t.Fatalf("ParseRequest() error = %v", err)
			}

			if request.Command != test.want {
				t.Errorf(
					"Command = %q, want %q",
					request.Command,
					test.want,
				)
			}
		})
	}
}

func TestParseRequestPreservesRawArgs(t *testing.T) {
	input := "CHAT   GLOBAL   Hola, mundo!  "
	request, err := protocol.ParseRequest(input)
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	if request.Command != "CHAT" {
		t.Errorf("Command = %q, want %q", request.Command, "CHAT")
	}

	if request.RawArgs != "GLOBAL   Hola, mundo!" {
		t.Errorf(
			"RawArgs = %q, want %q",
			request.RawArgs,
			"GLOBAL   Hola, mundo!",
		)
	}

	wantArgs := []string{"GLOBAL", "Hola,", "mundo!"}
	if len(request.Args) != len(wantArgs) {
		t.Fatalf(
			"len(Args) = %d, want %d; Args = %#v",
			len(request.Args),
			len(wantArgs),
			request.Args,
		)
	}

	for index, want := range wantArgs {
		if request.Args[index] != want {
			t.Errorf(
				"Args[%d] = %q, want %q",
				index,
				request.Args[index],
				want,
			)
		}
	}
}

func TestParseRequestWithoutArguments(t *testing.T) {
	request, err := protocol.ParseRequest("  look  ")
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	if request.Command != "LOOK" {
		t.Errorf("Command = %q, want %q", request.Command, "LOOK")
	}

	if len(request.Args) != 0 {
		t.Errorf("Args = %#v, want no arguments", request.Args)
	}

	if request.RawArgs != "" {
		t.Errorf("RawArgs = %q, want empty string", request.RawArgs)
	}
}

func TestParseRequestSplitsArgumentsOnWhitespace(t *testing.T) {
	request, err := protocol.ParseRequest("MOVE\t north")
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	if request.Command != "MOVE" {
		t.Errorf("Command = %q, want %q", request.Command, "MOVE")
	}

	if len(request.Args) != 1 || request.Args[0] != "north" {
		t.Errorf("Args = %#v, want []string{\"north\"}", request.Args)
	}

	if request.RawArgs != "north" {
		t.Errorf("RawArgs = %q, want %q", request.RawArgs, "north")
	}
}
