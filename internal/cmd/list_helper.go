package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	threads "github.com/salmonumbrella/threads-go"
	"github.com/salmonumbrella/threads-go/internal/iocontext"
	"github.com/salmonumbrella/threads-go/internal/outfmt"
)

// ListResult is a generic struct for paginated list results
type ListResult[T any] struct {
	Items   []T
	HasMore bool
	Cursor  string
}

// ListConfig is a generic struct for configuring list commands
type ListConfig[T any] struct {
	Use          string
	Short        string
	Long         string
	Example      string
	Headers      []string
	RowFunc      func(T) []string
	ColumnTypes  []outfmt.ColumnType
	EmptyMessage string

	// Fetch function - called with cursor and limit
	Fetch func(ctx context.Context, client *threads.Client, cursor string, limit int) (ListResult[T], error)
}

// NewListCommand creates a new list command using the provided configuration
func NewListCommand[T any](cfg ListConfig[T], getClient func(context.Context) (*threads.Client, error)) *cobra.Command {
	var limit int
	var cursor string

	cmd := &cobra.Command{
		Use:     cfg.Use,
		Short:   cfg.Short,
		Long:    cfg.Long,
		Example: cfg.Example,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			io := iocontext.GetIO(ctx)

			// Cap limit at 100
			if limit > 100 {
				limit = 100
			}

			// Default limit if not specified
			if limit == 0 {
				limit = 25
			}

			// Get client
			client, err := getClient(ctx)
			if err != nil {
				return err
			}

			// Fetch items
			result, err := cfg.Fetch(ctx, client, cursor, limit)
			if err != nil {
				return err
			}

			// Handle JSON output mode
			if outfmt.IsJSON(ctx) {
				return outputListJSON(io, result, cursor)
			}

			// Handle empty results in text mode
			if len(result.Items) == 0 {
				fmt.Fprintln(io.Out, cfg.EmptyMessage)
				return nil
			}

			// Build rows using RowFunc
			rows := make([][]string, len(result.Items))
			for i, item := range result.Items {
				rows[i] = cfg.RowFunc(item)
			}

			// Output table
			f := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
			if err := f.Table(cfg.Headers, rows, cfg.ColumnTypes); err != nil {
				return err
			}

			// Show pagination hint on stderr if there are more results
			if result.HasMore && result.Cursor != "" {
				fmt.Fprintf(io.ErrOut, "\nMore results available. Use --cursor %s to see next page.\n", result.Cursor)
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum number of results (1-100)")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor for next page")

	return cmd
}

// listJSONOutput is the JSON output structure for list commands
type listJSONOutput struct {
	Items   any    `json:"items"`
	HasMore bool   `json:"has_more"`
	Cursor  string `json:"cursor,omitempty"`
}

// outputListJSON outputs the list result as JSON
func outputListJSON[T any](io *iocontext.IO, result ListResult[T], requestCursor string) error {
	output := listJSONOutput{
		Items:   result.Items,
		HasMore: result.HasMore,
		Cursor:  result.Cursor,
	}

	// Handle empty items - ensure it's an empty array, not null
	if len(result.Items) == 0 {
		output.Items = []T{}
	}

	enc := json.NewEncoder(io.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}
