package ui

import (
	"fmt"
	"os"

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

// IsTerminal checks if stdout is a terminal
func IsTerminal() bool {
	return output.Profile != termenv.Ascii
}
