package outfmt

import (
	"bytes"
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
	if GetFormat(ctx) != Text {
		t.Errorf("default format should be Text")
	}

	// Set to JSON
	ctx = NewContext(ctx, JSON)
	if GetFormat(ctx) != JSON {
		t.Errorf("format should be JSON after setting")
	}

	// Set to Text
	ctx = NewContext(ctx, Text)
	if GetFormat(ctx) != Text {
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

func TestWithFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected Format
	}{
		{"json", JSON},
		{"text", Text},
		{"", Text},
		{"invalid", Text},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ctx := WithFormat(context.Background(), tt.input)
			if GetFormat(ctx) != tt.expected {
				t.Errorf("WithFormat(%q) = %v, want %v", tt.input, GetFormat(ctx), tt.expected)
			}
		})
	}
}

func TestIsJSON(t *testing.T) {
	ctx := context.Background()
	if IsJSON(ctx) {
		t.Error("default context should not be JSON")
	}

	ctx = WithFormat(ctx, "json")
	if !IsJSON(ctx) {
		t.Error("context with json format should return true for IsJSON")
	}

	ctx = WithFormat(ctx, "text")
	if IsJSON(ctx) {
		t.Error("context with text format should return false for IsJSON")
	}
}

func TestWithQuery(t *testing.T) {
	ctx := context.Background()
	if GetQuery(ctx) != "" {
		t.Error("default query should be empty")
	}

	ctx = WithQuery(ctx, ".field")
	if GetQuery(ctx) != ".field" {
		t.Errorf("query should be '.field', got %q", GetQuery(ctx))
	}
}

func TestWithYes(t *testing.T) {
	ctx := context.Background()
	if GetYes(ctx) {
		t.Error("default yes should be false")
	}

	ctx = WithYes(ctx, true)
	if !GetYes(ctx) {
		t.Error("yes should be true after setting")
	}

	ctx = WithYes(ctx, false)
	if GetYes(ctx) {
		t.Error("yes should be false after unsetting")
	}
}

func TestWithLimit(t *testing.T) {
	ctx := context.Background()
	if GetLimit(ctx) != 0 {
		t.Error("default limit should be 0")
	}

	ctx = WithLimit(ctx, 25)
	if GetLimit(ctx) != 25 {
		t.Errorf("limit should be 25, got %d", GetLimit(ctx))
	}
}

func TestFormatter_Table(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithFormat(context.Background(), "text")
	f := FromContext(ctx, WithWriter(&buf))

	headers := []string{"ID", "STATUS", "COUNT"}
	rows := [][]string{
		{"123", "ACTIVE", "10"},
		{"456", "PENDING", "20"},
	}

	if err := f.Table(headers, rows, nil); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "ID") {
		t.Error("missing header ID")
	}
	if !strings.Contains(output, "STATUS") {
		t.Error("missing header STATUS")
	}
	if !strings.Contains(output, "COUNT") {
		t.Error("missing header COUNT")
	}
	if !strings.Contains(output, "123") {
		t.Error("missing row data 123")
	}
	if !strings.Contains(output, "ACTIVE") {
		t.Error("missing row data ACTIVE")
	}
	if !strings.Contains(output, "456") {
		t.Error("missing row data 456")
	}
	if !strings.Contains(output, "PENDING") {
		t.Error("missing row data PENDING")
	}
}

func TestFormatter_Table_JSON(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithFormat(context.Background(), "json")
	f := FromContext(ctx, WithWriter(&buf))

	headers := []string{"ID", "STATUS", "COUNT"}
	rows := [][]string{
		{"123", "ACTIVE", "10"},
		{"456", "PENDING", "20"},
	}

	if err := f.Table(headers, rows, nil); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	// Should be JSON array
	if !strings.Contains(output, "[") {
		t.Error("JSON output should contain array bracket")
	}
	if !strings.Contains(output, `"ID"`) {
		t.Error("JSON should contain ID key")
	}
	if !strings.Contains(output, `"123"`) {
		t.Error("JSON should contain value 123")
	}
	if !strings.Contains(output, `"STATUS"`) {
		t.Error("JSON should contain STATUS key")
	}
	if !strings.Contains(output, `"ACTIVE"`) {
		t.Error("JSON should contain ACTIVE value")
	}
}

func TestFormatter_Table_WithColumnTypes(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithFormat(context.Background(), "text")
	f := FromContext(ctx, WithWriter(&buf))

	headers := []string{"ID", "STATUS", "AMOUNT"}
	rows := [][]string{
		{"123", "ACTIVE", "100.00"},
	}
	colTypes := []ColumnType{ColumnID, ColumnStatus, ColumnAmount}

	if err := f.Table(headers, rows, colTypes); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	// Since we're writing to a buffer (not TTY), colors should be disabled
	// Just verify the data is present
	if !strings.Contains(output, "123") {
		t.Error("missing ID value")
	}
	if !strings.Contains(output, "ACTIVE") {
		t.Error("missing STATUS value")
	}
	if !strings.Contains(output, "100.00") {
		t.Error("missing AMOUNT value")
	}
}

func TestFormatter_Output(t *testing.T) {
	t.Run("text mode", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := WithFormat(context.Background(), "text")
		f := FromContext(ctx, WithWriter(&buf))

		if err := f.Output("hello world"); err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(buf.String(), "hello world") {
			t.Error("output should contain message")
		}
	})

	t.Run("json mode", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := WithFormat(context.Background(), "json")
		f := FromContext(ctx, WithWriter(&buf))

		data := map[string]string{"key": "value"}
		if err := f.Output(data); err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(buf.String(), `"key"`) {
			t.Error("JSON output should contain key")
		}
		if !strings.Contains(buf.String(), `"value"`) {
			t.Error("JSON output should contain value")
		}
	})
}

func TestFormatter_Empty(t *testing.T) {
	t.Run("text mode", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := WithFormat(context.Background(), "text")
		f := FromContext(ctx, WithWriter(&buf))

		f.Empty("No results found")

		if !strings.Contains(buf.String(), "No results found") {
			t.Error("empty message should contain custom text")
		}
	})

	t.Run("json mode", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := WithFormat(context.Background(), "json")
		f := FromContext(ctx, WithWriter(&buf))

		f.Empty("No results found")

		if !strings.Contains(buf.String(), "[]") {
			t.Error("JSON empty should output empty array")
		}
	})
}

func TestFromContext_WithWriter(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	f := FromContext(ctx, WithWriter(&buf))

	f.Header("COL1", "COL2")
	f.Row("val1", "val2")
	f.Flush()

	output := buf.String()
	if !strings.Contains(output, "COL1") {
		t.Error("should write to custom writer")
	}
	if !strings.Contains(output, "val1") {
		t.Error("should write row data to custom writer")
	}
}

func TestFormatter_Table_JSONWithQuery(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithFormat(context.Background(), "json")
	ctx = WithQuery(ctx, ".[0].ID")
	f := FromContext(ctx, WithWriter(&buf))

	headers := []string{"ID", "STATUS"}
	rows := [][]string{
		{"123", "ACTIVE"},
		{"456", "PENDING"},
	}

	if err := f.Table(headers, rows, nil); err != nil {
		t.Fatal(err)
	}

	output := strings.TrimSpace(buf.String())
	// Should just output "123" (the ID of first row)
	if output != `"123"` {
		t.Errorf("expected \"123\", got %q", output)
	}
}
