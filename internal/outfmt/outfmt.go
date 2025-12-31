package outfmt

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/itchyny/gojq"
)

// Format represents output format type
type Format int

const (
	Text Format = iota
	JSON
)

type contextKey string

const formatKey contextKey = "output_format"

// ColumnType defines how a column should be formatted
type ColumnType int

const (
	ColumnPlain ColumnType = iota
	ColumnStatus
	ColumnAmount
	ColumnCurrency
	ColumnDate
	ColumnID
)

// ANSI escape codes for colors
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

// formatColumn applies formatting based on column type
func formatColumn(value string, colType ColumnType, colorEnabled bool) string {
	if !colorEnabled {
		return value
	}

	switch colType {
	case ColumnStatus:
		return formatStatus(value)
	case ColumnAmount:
		return formatAmount(value)
	case ColumnCurrency:
		return formatCurrency(value)
	case ColumnDate:
		return formatDate(value)
	case ColumnID:
		return formatID(value)
	default:
		return value
	}
}

// formatStatus colors status values based on their meaning
func formatStatus(status string) string {
	switch status {
	case "PUBLISHED", "ACTIVE", "FINISHED", "COMPLETED", "SUCCESS":
		return colorGreen + status + colorReset
	case "IN_PROGRESS", "PUBLISHING", "PENDING", "PROCESSING":
		return colorYellow + status + colorReset
	case "FAILED", "ERROR", "CANCELLED", "REJECTED":
		return colorRed + status + colorReset
	default:
		return status
	}
}

// formatAmount colors amounts based on sign (negative=red, positive=green)
func formatAmount(amount string) string {
	if len(amount) == 0 {
		return amount
	}
	if amount[0] == '-' {
		return colorRed + amount + colorReset
	}
	return colorGreen + amount + colorReset
}

// formatCurrency colors currency codes in cyan
func formatCurrency(currency string) string {
	return colorCyan + currency + colorReset
}

// formatDate colors dates in gray
func formatDate(date string) string {
	return colorGray + date + colorReset
}

// formatID colors IDs in blue
func formatID(id string) string {
	return colorBlue + id + colorReset
}

// NewContext creates a context with output format
func NewContext(ctx context.Context, format Format) context.Context {
	return context.WithValue(ctx, formatKey, format)
}

// FromContext gets output format from context
func FromContext(ctx context.Context) Format {
	if f, ok := ctx.Value(formatKey).(Format); ok {
		return f
	}
	return Text
}

// Output writes data in the appropriate format
func Output(ctx context.Context, data any, textFormatter func()) error {
	format := FromContext(ctx)
	switch format {
	case JSON:
		return WriteJSON(data, "")
	default:
		textFormatter()
		return nil
	}
}

// WriteJSON outputs JSON, optionally filtered by JQ query
func WriteJSON(data any, query string) error {
	if query != "" {
		return writeFilteredJSON(data, query)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func writeFilteredJSON(data any, query string) error {
	q, err := gojq.Parse(query)
	if err != nil {
		return fmt.Errorf("invalid jq query: %w", err)
	}

	code, err := gojq.Compile(q)
	if err != nil {
		return fmt.Errorf("failed to compile jq query: %w", err)
	}

	// Convert data to interface{} for gojq
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	var input any
	if err := json.Unmarshal(jsonBytes, &input); err != nil {
		return err
	}

	iter := code.Run(input)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return err
		}
		if err := enc.Encode(v); err != nil {
			return err
		}
	}
	return nil
}

// Formatter provides tabular text output
type Formatter struct {
	w *tabwriter.Writer
}

// NewFormatter creates a new text formatter
func NewFormatter() *Formatter {
	return &Formatter{
		w: tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0),
	}
}

// Header writes a header row
func (f *Formatter) Header(cols ...string) {
	for i, col := range cols {
		if i > 0 {
			fmt.Fprint(f.w, "\t")
		}
		fmt.Fprint(f.w, col)
	}
	fmt.Fprintln(f.w)
}

// Row writes a data row
func (f *Formatter) Row(cols ...any) {
	for i, col := range cols {
		if i > 0 {
			fmt.Fprint(f.w, "\t")
		}
		fmt.Fprint(f.w, col)
	}
	fmt.Fprintln(f.w)
}

// Flush writes all buffered output
func (f *Formatter) Flush() {
	f.w.Flush()
}

// Print outputs a simple message
func Print(format string, args ...any) {
	fmt.Printf(format, args...)
}

// Println outputs a line
func Println(args ...any) {
	fmt.Println(args...)
}
