package outfmt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/itchyny/gojq"
	"golang.org/x/term"
)

// Format represents output format type
type Format int

const (
	Text Format = iota
	JSON
)

type contextKey string

const (
	formatKey contextKey = "output_format"
	queryKey  contextKey = "output_query"
	yesKey    contextKey = "yes_flag"
	limitKey  contextKey = "limit_flag"
)

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

// WithFormat adds output format to context (string-based for CLI flags)
func WithFormat(ctx context.Context, format string) context.Context {
	switch format {
	case "json":
		return context.WithValue(ctx, formatKey, JSON)
	default:
		return context.WithValue(ctx, formatKey, Text)
	}
}

// WithQuery adds JQ query to context
func WithQuery(ctx context.Context, query string) context.Context {
	return context.WithValue(ctx, queryKey, query)
}

// GetQuery retrieves JQ query from context
func GetQuery(ctx context.Context) string {
	if q, ok := ctx.Value(queryKey).(string); ok {
		return q
	}
	return ""
}

// WithYes adds yes flag to context (for skipping confirmations)
func WithYes(ctx context.Context, yes bool) context.Context {
	return context.WithValue(ctx, yesKey, yes)
}

// GetYes retrieves yes flag from context
func GetYes(ctx context.Context) bool {
	if y, ok := ctx.Value(yesKey).(bool); ok {
		return y
	}
	return false
}

// WithLimit adds limit to context
func WithLimit(ctx context.Context, limit int) context.Context {
	return context.WithValue(ctx, limitKey, limit)
}

// GetLimit retrieves limit from context
func GetLimit(ctx context.Context) int {
	if l, ok := ctx.Value(limitKey).(int); ok {
		return l
	}
	return 0
}

// GetFormat retrieves format from context
func GetFormat(ctx context.Context) Format {
	if f, ok := ctx.Value(formatKey).(Format); ok {
		return f
	}
	return Text
}

// IsJSON checks if context has JSON output format
func IsJSON(ctx context.Context) bool {
	return GetFormat(ctx) == JSON
}

// Output writes data in the appropriate format (legacy, use Formatter.Output instead)
func Output(ctx context.Context, data any, textFormatter func()) error {
	format := GetFormat(ctx)
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

// OutputOption configures the Formatter
type OutputOption func(*Formatter)

// WithWriter sets a custom writer for output
func WithWriter(w io.Writer) OutputOption {
	return func(f *Formatter) {
		f.out = w
		f.w = tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	}
}

// Formatter provides tabular text output
type Formatter struct {
	ctx context.Context
	out io.Writer
	w   *tabwriter.Writer
}

// NewFormatter creates a new text formatter (legacy, use FromContext instead)
func NewFormatter() *Formatter {
	return &Formatter{
		ctx: context.Background(),
		out: os.Stdout,
		w:   tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0),
	}
}

// FromContext creates a Formatter from context with optional options
func FromContext(ctx context.Context, opts ...OutputOption) *Formatter {
	f := &Formatter{
		ctx: ctx,
		out: os.Stdout,
		w:   tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0),
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
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

// colorEnabled checks if color output is enabled
// Color is disabled if NO_COLOR env is set or stdout is not a TTY
func (f *Formatter) colorEnabled() bool {
	// Check NO_COLOR environment variable (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check if output is a TTY
	if file, ok := f.out.(*os.File); ok {
		return term.IsTerminal(int(file.Fd()))
	}

	// Non-file writers (like bytes.Buffer in tests) - disable color
	return false
}

// Table outputs data in tabular format with optional column colorization
func (f *Formatter) Table(headers []string, rows [][]string, colTypes []ColumnType) error {
	// In JSON mode, output as array of objects
	if IsJSON(f.ctx) {
		return f.tableJSON(headers, rows)
	}

	// Text mode - use tabwriter
	return f.tableText(headers, rows, colTypes)
}

// tableJSON outputs table data as JSON array of objects
func (f *Formatter) tableJSON(headers []string, rows [][]string) error {
	var result []map[string]string
	for _, row := range rows {
		obj := make(map[string]string)
		for i, header := range headers {
			if i < len(row) {
				obj[header] = row[i]
			}
		}
		result = append(result, obj)
	}

	query := GetQuery(f.ctx)
	if query != "" {
		return f.writeFilteredJSONTo(result, query)
	}

	enc := json.NewEncoder(f.out)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// writeFilteredJSONTo applies JQ filter and writes to output
func (f *Formatter) writeFilteredJSONTo(data any, query string) error {
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
	enc := json.NewEncoder(f.out)
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

// tableText outputs table data in aligned text format
func (f *Formatter) tableText(headers []string, rows [][]string, colTypes []ColumnType) error {
	colorOn := f.colorEnabled()

	// Write headers
	for i, header := range headers {
		if i > 0 {
			fmt.Fprint(f.w, "\t")
		}
		fmt.Fprint(f.w, header)
	}
	fmt.Fprintln(f.w)

	// Write rows with optional colorization
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(f.w, "\t")
			}
			// Apply column type formatting if provided
			if colTypes != nil && i < len(colTypes) {
				cell = formatColumn(cell, colTypes[i], colorOn)
			}
			fmt.Fprint(f.w, cell)
		}
		fmt.Fprintln(f.w)
	}

	return f.w.Flush()
}

// Output writes data in the appropriate format (JSON or pretty-print)
func (f *Formatter) Output(data any) error {
	if IsJSON(f.ctx) {
		query := GetQuery(f.ctx)
		if query != "" {
			return f.writeFilteredJSONTo(data, query)
		}
		enc := json.NewEncoder(f.out)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}

	// For text output, just print the value
	fmt.Fprintln(f.out, data)
	return nil
}

// Empty prints an empty result message
func (f *Formatter) Empty(msg string) {
	if IsJSON(f.ctx) {
		enc := json.NewEncoder(f.out)
		enc.SetIndent("", "  ")
		enc.Encode([]any{})
		return
	}
	fmt.Fprintln(f.out, msg)
}

// Print outputs a simple message
func Print(format string, args ...any) {
	fmt.Printf(format, args...)
}

// Println outputs a line
func Println(args ...any) {
	fmt.Println(args...)
}
