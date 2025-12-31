package outfmt

import (
	"context"
	"strings"
	"testing"
)

func TestColumnTypes(t *testing.T) {
	tests := []struct {
		name     string
		colType  ColumnType
		value    string
		wantDiff bool // whether output differs from input (colorized)
	}{
		{"plain stays same", ColumnPlain, "test", false},
		{"status COMPLETED gets color", ColumnStatus, "COMPLETED", true},
		{"status ACTIVE gets color", ColumnStatus, "ACTIVE", true},
		{"status IN_PROGRESS gets color", ColumnStatus, "IN_PROGRESS", true},
		{"status FAILED gets color", ColumnStatus, "FAILED", true},
		{"status unknown stays same", ColumnStatus, "UNKNOWN", false},
		{"amount positive gets color", ColumnAmount, "100.00", true},
		{"amount negative gets color", ColumnAmount, "-50.00", true},
		{"amount empty stays same", ColumnAmount, "", false},
		{"currency gets color", ColumnCurrency, "USD", true},
		{"date gets color", ColumnDate, "2024-01-15", true},
		{"id gets color", ColumnID, "abc123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Color disabled - should never change
			result := formatColumn(tt.value, tt.colType, false)
			if result != tt.value {
				t.Errorf("with color disabled, got %q want %q", result, tt.value)
			}

			// Color enabled - should change if wantDiff is true
			resultColor := formatColumn(tt.value, tt.colType, true)
			hasDiff := resultColor != tt.value
			if hasDiff != tt.wantDiff {
				t.Errorf("with color enabled, got diff=%v want diff=%v (result=%q)", hasDiff, tt.wantDiff, resultColor)
			}
		})
	}
}

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		status        string
		expectedColor string
	}{
		{"PUBLISHED", colorGreen},
		{"ACTIVE", colorGreen},
		{"FINISHED", colorGreen},
		{"COMPLETED", colorGreen},
		{"SUCCESS", colorGreen},
		{"IN_PROGRESS", colorYellow},
		{"PUBLISHING", colorYellow},
		{"PENDING", colorYellow},
		{"PROCESSING", colorYellow},
		{"FAILED", colorRed},
		{"ERROR", colorRed},
		{"CANCELLED", colorRed},
		{"REJECTED", colorRed},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := formatStatus(tt.status)
			if !strings.HasPrefix(result, tt.expectedColor) {
				t.Errorf("formatStatus(%q) = %q, expected to start with %q", tt.status, result, tt.expectedColor)
			}
			if !strings.HasSuffix(result, colorReset) {
				t.Errorf("formatStatus(%q) = %q, expected to end with reset code", tt.status, result)
			}
			if !strings.Contains(result, tt.status) {
				t.Errorf("formatStatus(%q) = %q, expected to contain original status", tt.status, result)
			}
		})
	}
}

func TestFormatAmount(t *testing.T) {
	tests := []struct {
		amount        string
		expectedColor string
	}{
		{"100.00", colorGreen},
		{"0.01", colorGreen},
		{"-50.00", colorRed},
		{"-0.01", colorRed},
	}

	for _, tt := range tests {
		t.Run(tt.amount, func(t *testing.T) {
			result := formatAmount(tt.amount)
			if !strings.HasPrefix(result, tt.expectedColor) {
				t.Errorf("formatAmount(%q) = %q, expected to start with %q", tt.amount, result, tt.expectedColor)
			}
			if !strings.HasSuffix(result, colorReset) {
				t.Errorf("formatAmount(%q) = %q, expected to end with reset code", tt.amount, result)
			}
		})
	}
}

func TestFormatAmountEmpty(t *testing.T) {
	result := formatAmount("")
	if result != "" {
		t.Errorf("formatAmount(\"\") = %q, expected empty string", result)
	}
}

func TestFormatCurrency(t *testing.T) {
	result := formatCurrency("USD")
	if !strings.HasPrefix(result, colorCyan) {
		t.Errorf("formatCurrency(\"USD\") should start with cyan color")
	}
	if !strings.Contains(result, "USD") {
		t.Errorf("formatCurrency(\"USD\") should contain USD")
	}
	if !strings.HasSuffix(result, colorReset) {
		t.Errorf("formatCurrency(\"USD\") should end with reset")
	}
}

func TestFormatDate(t *testing.T) {
	result := formatDate("2024-01-15")
	if !strings.HasPrefix(result, colorGray) {
		t.Errorf("formatDate should start with gray color")
	}
	if !strings.Contains(result, "2024-01-15") {
		t.Errorf("formatDate should contain the date")
	}
	if !strings.HasSuffix(result, colorReset) {
		t.Errorf("formatDate should end with reset")
	}
}

func TestFormatID(t *testing.T) {
	result := formatID("abc123")
	if !strings.HasPrefix(result, colorBlue) {
		t.Errorf("formatID should start with blue color")
	}
	if !strings.Contains(result, "abc123") {
		t.Errorf("formatID should contain the ID")
	}
	if !strings.HasSuffix(result, colorReset) {
		t.Errorf("formatID should end with reset")
	}
}

func TestFormat(t *testing.T) {
	if Text != 0 {
		t.Errorf("Text format should be 0, got %d", Text)
	}
	if JSON != 1 {
		t.Errorf("JSON format should be 1, got %d", JSON)
	}
}

func TestContextFormat(t *testing.T) {
	ctx := context.Background()

	// Default should be Text
	if FromContext(ctx) != Text {
		t.Errorf("default format should be Text")
	}

	// Set to JSON
	ctx = NewContext(ctx, JSON)
	if FromContext(ctx) != JSON {
		t.Errorf("format should be JSON after setting")
	}

	// Set to Text
	ctx = NewContext(ctx, Text)
	if FromContext(ctx) != Text {
		t.Errorf("format should be Text after setting")
	}
}

func TestColumnTypeConstants(t *testing.T) {
	// Verify column types are distinct
	types := []ColumnType{ColumnPlain, ColumnStatus, ColumnAmount, ColumnCurrency, ColumnDate, ColumnID}
	seen := make(map[ColumnType]bool)
	for _, ct := range types {
		if seen[ct] {
			t.Errorf("duplicate column type value: %d", ct)
		}
		seen[ct] = true
	}

	// Verify ColumnPlain is 0 (iota default)
	if ColumnPlain != 0 {
		t.Errorf("ColumnPlain should be 0, got %d", ColumnPlain)
	}
}
