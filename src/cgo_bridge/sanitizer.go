package cgo_bridge

import (
	"strings"
)

// SanitizeString ensures that strings coming from C are clean.
func SanitizeString(input string) string {
	s := strings.TrimSpace(input)
	s = strings.ReplaceAll(s, "\x00", "")
	return s
}
