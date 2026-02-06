package ui

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/muesli/termenv"

	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

// Printer provides styled, context-aware output.
type Printer struct {
	out    io.Writer
	errOut io.Writer
	output *termenv.Output

	Green  termenv.Color
	Red    termenv.Color
	Yellow termenv.Color
	Blue   termenv.Color
	Gray   termenv.Color
	Cyan   termenv.Color
}

// New creates a Printer for the given IO and color mode.
func New(io *iocontext.IO, colorMode outfmt.ColorMode) *Printer {
	out := io.Out
	errOut := io.ErrOut
	return NewWithWriters(out, errOut, colorMode)
}

// NewWithWriters creates a Printer using explicit writers.
// This is useful when you want to route status/progress output to stderr (e.g. in JSON mode)
// while keeping command data on stdout.
func NewWithWriters(out io.Writer, errOut io.Writer, colorMode outfmt.ColorMode) *Printer {
	if out == nil {
		out = os.Stdout
	}
	if errOut == nil {
		errOut = os.Stderr
	}

	opts := []termenv.OutputOption{}
	switch colorMode {
	case outfmt.ColorAlways:
		opts = append(opts, termenv.WithUnsafe())
	case outfmt.ColorNever:
		opts = append(opts, termenv.WithProfile(termenv.Ascii))
	}

	output := termenv.NewOutput(out, opts...)

	return &Printer{
		out:    out,
		errOut: errOut,
		output: output,
		Green:  output.Color("#22c55e"),
		Red:    output.Color("#ef4444"),
		Yellow: output.Color("#eab308"),
		Blue:   output.Color("#3b82f6"),
		Gray:   output.Color("#6b7280"),
		Cyan:   output.Color("#06b6d4"),
	}
}

// Success prints a success message.
func (p *Printer) Success(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(p.out, p.output.String("✓ ").Foreground(p.Green).String()+msg) //nolint:errcheck // Best-effort output
}

// Error prints an error message.
func (p *Printer) Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(p.errOut, p.output.String("✗ ").Foreground(p.Red).String()+msg) //nolint:errcheck // Best-effort output
}

// Warning prints a warning message.
func (p *Printer) Warning(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(p.out, p.output.String("⚠ ").Foreground(p.Yellow).String()+msg) //nolint:errcheck // Best-effort output
}

// Info prints an info message.
func (p *Printer) Info(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(p.out, p.output.String("ℹ ").Foreground(p.Blue).String()+msg) //nolint:errcheck // Best-effort output
}

// Bold returns bold text.
func (p *Printer) Bold(s string) string {
	return p.output.String(s).Bold().String()
}

// Dim returns dimmed text.
func (p *Printer) Dim(s string) string {
	return p.output.String(s).Faint().String()
}

// Colorize returns colored text.
func (p *Printer) Colorize(s string, color termenv.Color) string {
	return p.output.String(s).Foreground(color).String()
}

// StatusColor returns appropriate color for a status.
func (p *Printer) StatusColor(status string) termenv.Color {
	switch status {
	case "active", "valid", "published", "success":
		return p.Green
	case "expired", "error", "failed":
		return p.Red
	case "pending", "processing":
		return p.Yellow
	default:
		return p.Gray
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
	output := termenv.NewOutput(os.Stdout)
	return output.Profile != termenv.Ascii
}
