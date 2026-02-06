package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/threads-cli/internal/api"
	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
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
	Fetch func(ctx context.Context, client *api.Client, cursor string, limit int) (ListResult[T], error)
}

// NewListCommand creates a new list command using the provided configuration
func NewListCommand[T any](cfg ListConfig[T], getClient func(context.Context) (*api.Client, error)) *cobra.Command {
	var limit int
	var cursor string
	var noHints bool

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

			// Handle JSONL output mode (one item per line).
			if outfmt.IsJSONL(ctx) {
				out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
				if err := out.Output(result.Items); err != nil {
					return err
				}
				if !noHints && result.HasMore && result.Cursor != "" {
					fmt.Fprintf(io.ErrOut, "\nMore results available. Use --cursor %s to see next page.\n", result.Cursor) //nolint:errcheck // Best-effort output to stderr
				}
				return nil
			}

			// Handle JSON output mode
			if outfmt.IsJSON(ctx) {
				return outputListJSON(io, result, cursor, outfmt.GetQuery(ctx))
			}

			// Handle empty results in text mode
			if len(result.Items) == 0 {
				fmt.Fprintln(io.Out, cfg.EmptyMessage) //nolint:errcheck // Best-effort output to stdout
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
			if !noHints && result.HasMore && result.Cursor != "" {
				fmt.Fprintf(io.ErrOut, "\nMore results available. Use --cursor %s to see next page.\n", result.Cursor) //nolint:errcheck // Best-effort output to stderr
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum number of results (1-100)")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor for next page")
	cmd.Flags().BoolVar(&noHints, "no-hints", false, "Suppress pagination hints on stderr")

	return cmd
}

// listJSONOutput is the JSON output structure for list commands
type listJSONOutput struct {
	Items   any    `json:"items"`
	HasMore bool   `json:"has_more"`
	Cursor  string `json:"cursor,omitempty"`
}

// outputListJSON outputs the list result as JSON
//
//nolint:unparam // requestCursor reserved for future pagination features
func outputListJSON[T any](io *iocontext.IO, result ListResult[T], _ string, query string) error {
	output := listJSONOutput{
		Items:   result.Items,
		HasMore: result.HasMore,
		Cursor:  result.Cursor,
	}

	// Handle empty items - ensure it's an empty array, not null
	if len(result.Items) == 0 {
		output.Items = []T{}
	}

	return outfmt.WriteJSONTo(io.Out, output, query)
}
