package protocol

import (
	"unicode"
	"unicode/utf8"
)

const MaxUsernameLength = 12

// ValidUsername accepts usernames containing Unicode letters and numbers,
// combining marks, and underscores, up to 24 Unicode code points.
func ValidUsername(username string) bool {
	if !utf8.ValidString(username) || username == "" ||
		utf8.RuneCountInString(username) > MaxUsernameLength {
		return false
	}

	hasBase := false
	for _, r := range username {
		switch {
		case unicode.IsLetter(r), unicode.IsNumber(r), r == '_':
			hasBase = true
		case unicode.IsMark(r) && hasBase:
		default:
			return false
		}
	}

	return true
}
