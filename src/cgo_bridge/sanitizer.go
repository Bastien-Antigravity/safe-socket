package cgo_bridge

// =============================================================================
// ESSENTIAL PROCESS:
// Sanitizes incoming C strings passed across the CGO boundary, trimming whitespace
// and stripping null terminators to prevent memory and string corruption in Go.
//
// DATA FLOW:
// 1. Input: Raw C string converted to Go string.
// 2. Logic: Trims surrounding whitespace and removes null byte characters ('\x00').
// 3. Output: Cleaned string safe for internal Go runtime processing.
//
// KEY PARAMETERS:
// - SanitizeString: Sanitization helper function.
// =============================================================================

import (
	"strings"
)

// SanitizeString ensures that strings coming from C are clean.
func SanitizeString(input string) string {
	s := strings.TrimSpace(input)
	s = strings.ReplaceAll(s, "\x00", "")
	return s
}
