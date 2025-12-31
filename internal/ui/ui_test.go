package ui

import (
	"testing"
	"time"
)

func TestFormatRelativeTime(t *testing.T) {
	// Use a fixed reference time for deterministic tests
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		offset   time.Duration
		expected string
	}{
		// Future times
		{"30 seconds from now", 30 * time.Second, "now"},
		{"5 minutes from now", 5 * time.Minute, "in 5m"},
		{"45 minutes from now", 45 * time.Minute, "in 45m"},
		{"1 hour from now", time.Hour, "in 1h"},
		{"1.5 hours from now", 90 * time.Minute, "in 1h 30m"},
		{"3 hours from now", 3 * time.Hour, "in 3h"},
		{"1 day from now", 24 * time.Hour, "in 1d"},
		{"1 day 12 hours from now", 36 * time.Hour, "in 1d 12h"},
		{"3 days from now", 72 * time.Hour, "in 3d"},
		{"1 week from now", 7 * 24 * time.Hour, "in 1w"},
		{"10 days from now", 10 * 24 * time.Hour, "in 1w 3d"},

		// Past times
		{"-30 seconds ago", -30 * time.Second, "now"},
		{"5 minutes ago", -5 * time.Minute, "5m ago"},
		{"45 minutes ago", -45 * time.Minute, "45m ago"},
		{"1 hour ago", -time.Hour, "1h ago"},
		{"1.5 hours ago", -90 * time.Minute, "1h 30m ago"},
		{"3 hours ago", -3 * time.Hour, "3h ago"},
		{"1 day ago", -24 * time.Hour, "1d ago"},
		{"1 day 12 hours ago", -36 * time.Hour, "1d 12h ago"},
		{"3 days ago", -72 * time.Hour, "3d ago"},
		{"1 week ago", -7 * 24 * time.Hour, "1w ago"},
		{"10 days ago", -10 * 24 * time.Hour, "1w 3d ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetTime := now.Add(tt.offset)
			result := formatRelativeTimeFrom(targetTime, now)
			if result != tt.expected {
				t.Errorf("formatRelativeTimeFrom(%v) = %q, want %q", tt.offset, result, tt.expected)
			}
		})
	}
}
