package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/muesli/termenv"
)

var (
	output = termenv.NewOutput(os.Stdout)
	// Colors
	Green  = output.Color("#22c55e")
	Red    = output.Color("#ef4444")
	Yellow = output.Color("#eab308")
	Blue   = output.Color("#3b82f6")
	Gray   = output.Color("#6b7280")
	Cyan   = output.Color("#06b6d4")
)

// Success prints a success message
func Success(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(output.String("✓ ").Foreground(Green).String() + msg)
}

// Error prints an error message
func Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, output.String("✗ ").Foreground(Red).String()+msg)
}

// Warning prints a warning message
func Warning(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(output.String("⚠ ").Foreground(Yellow).String() + msg)
}

// Info prints an info message
func Info(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(output.String("ℹ ").Foreground(Blue).String() + msg)
}

// Bold returns bold text
func Bold(s string) string {
	return output.String(s).Bold().String()
}

// Dim returns dimmed text
func Dim(s string) string {
	return output.String(s).Faint().String()
}

// Colorize returns colored text
func Colorize(s string, color termenv.Color) string {
	return output.String(s).Foreground(color).String()
}

// StatusColor returns appropriate color for a status
func StatusColor(status string) termenv.Color {
	switch status {
	case "active", "valid", "published", "success":
		return Green
	case "expired", "error", "failed":
		return Red
	case "pending", "processing":
		return Yellow
	default:
		return Gray
	}
}

// FormatDuration formats a duration in human-readable form
func FormatDuration(days float64) string {
	if days < 0 {
		return "unknown"
	}
	if days < 1 {
		hours := days * 24
		return fmt.Sprintf("%.0f hours", hours)
	}
	if days < 7 {
		return fmt.Sprintf("%.0f days", days)
	}
	weeks := days / 7
	return fmt.Sprintf("%.1f weeks", weeks)
}

// FormatRelativeTime formats a time relative to now (e.g., "in 2h 30m" or "expired 1h ago")
func FormatRelativeTime(t time.Time) string {
	return formatRelativeTimeFrom(t, time.Now())
}

// formatRelativeTimeFrom formats a time relative to a reference time (for testing)
func formatRelativeTimeFrom(t, now time.Time) string {
	d := t.Sub(now)

	// Handle the "now" case
	if d > -time.Minute && d < time.Minute {
		return "now"
	}

	prefix := "in "
	suffix := ""
	if d < 0 {
		prefix = ""
		suffix = " ago"
		d = -d
	}

	// Format the duration
	var result string
	if d < time.Hour {
		mins := int(d.Minutes())
		result = fmt.Sprintf("%dm", mins)
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		mins := int(d.Minutes()) % 60
		if mins > 0 {
			result = fmt.Sprintf("%dh %dm", hours, mins)
		} else {
			result = fmt.Sprintf("%dh", hours)
		}
	} else if d < 7*24*time.Hour {
		days := int(d.Hours() / 24)
		hours := int(d.Hours()) % 24
		if hours > 0 {
			result = fmt.Sprintf("%dd %dh", days, hours)
		} else {
			result = fmt.Sprintf("%dd", days)
		}
	} else {
		weeks := int(d.Hours() / (24 * 7))
		days := int(d.Hours()/24) % 7
		if days > 0 {
			result = fmt.Sprintf("%dw %dd", weeks, days)
		} else {
			result = fmt.Sprintf("%dw", weeks)
		}
	}

	return prefix + result + suffix
}

// IsTerminal checks if stdout is a terminal
func IsTerminal() bool {
	return output.Profile != termenv.Ascii
}
