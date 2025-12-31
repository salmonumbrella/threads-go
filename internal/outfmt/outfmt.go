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
