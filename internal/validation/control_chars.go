package validation

import (
	"unicode"
)

const (
	controlCharsTagName = "control_chars"
)

// ContainsControlChars reports whether s contains any control character,
// as defined by unicode.IsControl.
func ContainsControlChars(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
