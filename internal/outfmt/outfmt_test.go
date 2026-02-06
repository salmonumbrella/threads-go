package outfmt

import (
	"bytes"
	"context"
	"os"
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
	if JSONL != 2 {
		t.Errorf("JSONL format should be 2, got %d", JSONL)
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
		{"jsonl", JSONL},
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

	ctx = WithFormat(ctx, "jsonl")
	if !IsJSON(ctx) {
		t.Error("context with jsonl format should return true for IsJSON")
	}

	ctx = WithFormat(ctx, "text")
	if IsJSON(ctx) {
		t.Error("context with text format should return false for IsJSON")
	}
}

func TestIsJSONL(t *testing.T) {
	ctx := context.Background()
	if IsJSONL(ctx) {
		t.Error("default context should not be JSONL")
	}

	ctx = WithFormat(ctx, "json")
	if IsJSONL(ctx) {
		t.Error("json context should not be JSONL")
	}

	ctx = WithFormat(ctx, "jsonl")
	if !IsJSONL(ctx) {
		t.Error("jsonl context should be JSONL")
	}
}

func TestFormatterOutput_JSONL_Slice(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithFormat(context.Background(), "jsonl")
	f := FromContext(ctx, WithWriter(&buf))

	err := f.Output([]map[string]any{
		{"id": "1"},
		{"id": "2"},
	})
	if err != nil {
		t.Fatalf("Output jsonl: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), buf.String())
	}
	if !strings.Contains(lines[0], `"id"`) || !strings.Contains(lines[0], `"1"`) {
		t.Fatalf("unexpected first line: %q", lines[0])
	}
	if !strings.Contains(lines[1], `"id"`) || !strings.Contains(lines[1], `"2"`) {
		t.Fatalf("unexpected second line: %q", lines[1])
	}
}

func TestFormatterTable_JSONL(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithFormat(context.Background(), "jsonl")
	f := FromContext(ctx, WithWriter(&buf))

	err := f.Table([]string{"ID", "TEXT"}, [][]string{{"1", "Hello"}, {"2", "World"}}, nil)
	if err != nil {
		t.Fatalf("Table jsonl: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), buf.String())
	}
	if !strings.Contains(lines[0], `"ID"`) || !strings.Contains(lines[0], `"1"`) {
		t.Fatalf("unexpected first line: %q", lines[0])
	}
	if !strings.Contains(lines[1], `"ID"`) || !strings.Contains(lines[1], `"2"`) {
		t.Fatalf("unexpected second line: %q", lines[1])
	}
}

func TestFormatterEmpty_JSONL(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithFormat(context.Background(), "jsonl")
	f := FromContext(ctx, WithWriter(&buf))
	f.Empty("No results")
	if buf.String() != "" {
		t.Fatalf("expected no output for jsonl Empty, got %q", buf.String())
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

func TestFormatter_Output_JSONWithQuery(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithFormat(context.Background(), "json")
	ctx = WithQuery(ctx, ".key")
	f := FromContext(ctx, WithWriter(&buf))

	data := map[string]string{"key": "filtered_value"}
	if err := f.Output(data); err != nil {
		t.Fatal(err)
	}

	output := strings.TrimSpace(buf.String())
	if output != `"filtered_value"` {
		t.Errorf("expected \"filtered_value\", got %q", output)
	}
}

func TestNewFormatter(t *testing.T) {
	f := NewFormatter()
	if f == nil {
		t.Fatal("NewFormatter should return non-nil formatter")
	}
	if f.ctx == nil {
		t.Error("NewFormatter should initialize context")
	}
	if f.out == nil {
		t.Error("NewFormatter should initialize output writer")
	}
	if f.w == nil {
		t.Error("NewFormatter should initialize tabwriter")
	}
}

func TestOutput_PackageLevel(t *testing.T) {
	t.Run("text mode calls text formatter", func(t *testing.T) {
		ctx := WithFormat(context.Background(), "text")
		called := false
		err := Output(ctx, map[string]string{"key": "value"}, func() {
			called = true
		})
		if err != nil {
			t.Fatal(err)
		}
		if !called {
			t.Error("text formatter should be called in text mode")
		}
	})

	t.Run("json mode writes json", func(t *testing.T) {
		// This writes to stdout, so we just verify no error
		ctx := WithFormat(context.Background(), "json")
		called := false
		// Redirect stdout for this test
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := Output(ctx, map[string]string{"key": "value"}, func() {
			called = true
		})

		_ = w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)

		if err != nil {
			t.Fatal(err)
		}
		if called {
			t.Error("text formatter should NOT be called in json mode")
		}
		if !strings.Contains(buf.String(), "key") {
			t.Error("JSON output should contain key")
		}
	})
}

func TestWriteJSON(t *testing.T) {
	t.Run("without query", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := WriteJSON(map[string]string{"hello": "world"}, "")

		_ = w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)

		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "hello") {
			t.Error("JSON should contain hello")
		}
		if !strings.Contains(buf.String(), "world") {
			t.Error("JSON should contain world")
		}
	})

	t.Run("with valid query", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := WriteJSON(map[string]string{"hello": "world"}, ".hello")

		_ = w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)

		if err != nil {
			t.Fatal(err)
		}
		output := strings.TrimSpace(buf.String())
		if output != `"world"` {
			t.Errorf("expected \"world\", got %q", output)
		}
	})
}

func TestWriteFilteredJSON_Errors(t *testing.T) {
	t.Run("invalid query syntax", func(t *testing.T) {
		err := writeFilteredJSON(map[string]string{"a": "b"}, ".[invalid")
		if err == nil {
			t.Error("expected error for invalid query")
		}
		if !strings.Contains(err.Error(), "invalid jq query") {
			t.Errorf("error should mention invalid jq query, got: %v", err)
		}
	})

	t.Run("compile error", func(t *testing.T) {
		// Using a query that parses but fails to compile - use $undefined variable
		err := writeFilteredJSON(map[string]string{"a": "b"}, "$undefined")
		if err == nil {
			t.Error("expected error for undefined variable")
		}
		if !strings.Contains(err.Error(), "failed to compile") {
			t.Errorf("error should mention compile failure, got: %v", err)
		}
	})

	t.Run("runtime error in jq", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// This will cause a runtime error (accessing non-existent key with error)
		err := writeFilteredJSON(map[string]string{"a": "b"}, ".nonexistent | error")

		_ = w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)

		if err == nil {
			t.Error("expected error from jq runtime error")
		}
	})
}

func TestWriteFilteredJSON_MultipleResults(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Query that returns multiple values
	data := []map[string]string{{"id": "1"}, {"id": "2"}, {"id": "3"}}
	err := writeFilteredJSON(data, ".[].id")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, `"1"`) {
		t.Error("should contain first id")
	}
	if !strings.Contains(output, `"2"`) {
		t.Error("should contain second id")
	}
	if !strings.Contains(output, `"3"`) {
		t.Error("should contain third id")
	}
}

func TestPrint(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Print("hello %s", "world")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if buf.String() != "hello world" {
		t.Errorf("expected 'hello world', got %q", buf.String())
	}
}

func TestPrintln(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Println("hello", "world")

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if strings.TrimSpace(buf.String()) != "hello world" {
		t.Errorf("expected 'hello world', got %q", buf.String())
	}
}

func TestFormatter_WriteFilteredJSONTo_InvalidQuery(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithFormat(context.Background(), "json")
	ctx = WithQuery(ctx, ".[invalid")
	f := FromContext(ctx, WithWriter(&buf))

	headers := []string{"ID"}
	rows := [][]string{{"123"}}

	err := f.Table(headers, rows, nil)
	if err == nil {
		t.Error("expected error for invalid query")
	}
	if !strings.Contains(err.Error(), "invalid jq query") {
		t.Errorf("error should mention invalid jq query, got: %v", err)
	}
}

func TestFormatter_WriteFilteredJSONTo_CompileError(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithFormat(context.Background(), "json")
	ctx = WithQuery(ctx, "$undefined")
	f := FromContext(ctx, WithWriter(&buf))

	headers := []string{"ID"}
	rows := [][]string{{"123"}}

	err := f.Table(headers, rows, nil)
	if err == nil {
		t.Error("expected error for undefined variable")
	}
	if !strings.Contains(err.Error(), "failed to compile") {
		t.Errorf("error should mention compile failure, got: %v", err)
	}
}

func TestFormatter_WriteFilteredJSONTo_RuntimeError(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithFormat(context.Background(), "json")
	ctx = WithQuery(ctx, ".[0].ID | error")
	f := FromContext(ctx, WithWriter(&buf))

	headers := []string{"ID"}
	rows := [][]string{{"123"}}

	err := f.Table(headers, rows, nil)
	if err == nil {
		t.Error("expected error from jq runtime error")
	}
}

func TestFormatter_Output_InvalidQuery(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithFormat(context.Background(), "json")
	ctx = WithQuery(ctx, ".[invalid")
	f := FromContext(ctx, WithWriter(&buf))

	err := f.Output(map[string]string{"key": "value"})
	if err == nil {
		t.Error("expected error for invalid query")
	}
}

func TestColorEnabled_NoColorEnv(t *testing.T) {
	// Test that setting NO_COLOR disables color
	t.Setenv("NO_COLOR", "1")

	var buf bytes.Buffer
	ctx := WithFormat(context.Background(), "text")
	f := FromContext(ctx, WithWriter(&buf))

	// colorEnabled should return false due to NO_COLOR
	if f.colorEnabled() {
		t.Error("colorEnabled should return false when NO_COLOR is set")
	}
}

func TestColorEnabled_WithFile(t *testing.T) {
	// Create a temp file (not a TTY)
	tmpFile, err := os.CreateTemp("", "outfmt_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	defer func() { _ = tmpFile.Close() }()

	// Ensure NO_COLOR is not set
	t.Setenv("NO_COLOR", "")

	ctx := WithFormat(context.Background(), "text")
	f := FromContext(ctx, WithWriter(tmpFile))

	// colorEnabled should return false because file is not a TTY
	if f.colorEnabled() {
		t.Error("colorEnabled should return false for non-TTY file")
	}
}
