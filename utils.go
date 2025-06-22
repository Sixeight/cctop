package main

import (
	"fmt"
	"strings"
	"time"
)

// formatNumber formats a number with comma separators
func formatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}

	// Build result from right to left
	var result strings.Builder
	for i := len(s) - 1; i >= 0; i-- {
		if (len(s)-i-1)%3 == 0 && len(s)-i-1 > 0 {
			result.WriteString(",")
		}
		result.WriteByte(s[i])
	}

	// Reverse the string
	runes := []rune(result.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}

// formatTime formats minutes into a human-readable time string
func formatTime(minutes float64) string {
	if minutes < 0 {
		minutes = 0
	}

	if minutes < MinutesPerHour {
		return fmt.Sprintf("%dm", int(minutes))
	}

	hours := int(minutes / MinutesPerHour)
	mins := int(minutes) % int(MinutesPerHour)

	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}

	return fmt.Sprintf("%dh%dm", hours, mins)
}

// Terminal control functions
func hideCursor()   { fmt.Print(HideCursor) }
func showCursor()   { fmt.Print(ShowCursor) }
func clearScreen()  { fmt.Print(ClearScreen) }
func clearAndHome() { fmt.Print(ClearAndHome) }

// Time utility functions moved from burnrate.go
func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

// clampInt ensures an integer value is within the specified range
func clampInt(value, minVal, maxVal int) int {
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}
