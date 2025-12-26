package service

import (
	"fmt"
	"strconv"
)

// parseUint64 is a helper function to parse string to uint64 safely
func parseUint64(s string) uint64 {
	if s == "" {
		return 0
	}
	if val, err := strconv.ParseUint(s, 10, 64); err == nil {
		return val
	}
	// Log error for debugging but return 0 to prevent panic
	fmt.Printf("Warning: Failed to parse uint64 from string: '%s'\n", s)
	return 0
}

// CalculateDelta calculates the delta between current and previous values with protection against false resets
func CalculateDelta(current, previous uint64, isReset bool) uint64 {
	if isReset || current < previous {
		return current // If reset, use current value as new baseline
	}
	return current - previous
}
