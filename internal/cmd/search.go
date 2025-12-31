package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	threads "github.com/salmonumbrella/threads-go"
	"github.com/salmonumbrella/threads-go/internal/outfmt"
)

func newSearchCmd() *cobra.Command {
	var limit int
	var cursor string
	var mediaType string
	var since string
	var until string

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search posts by keyword",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			ctx := cmd.Context()

			client, err := getClient(ctx)
			if err != nil {
				return err
			}

			opts := &threads.SearchOptions{
				Limit: limit,
				After: cursor,
			}

			if mediaType != "" {
				opts.MediaType = mediaType
			}

			if since != "" {
				t, err := time.Parse("2006-01-02", since)
				if err != nil {
					return fmt.Errorf("invalid --since date: %w", err)
				}
				opts.Since = t.Unix()
			}

			if until != "" {
				t, err := time.Parse("2006-01-02", until)
				if err != nil {
					return fmt.Errorf("invalid --until date: %w", err)
				}
				opts.Until = t.Unix()
			}

			result, err := client.KeywordSearch(ctx, query, opts)
			if err != nil {
				return err
			}

			if outfmt.IsJSON(ctx) {
				return outfmt.WriteJSON(result, jqQuery)
			}

			f := outfmt.FromContext(ctx)

			if len(result.Data) == 0 {
				f.Empty("No results found")
				return nil
			}

			headers := []string{"ID", "USER", "TEXT", "TYPE", "DATE"}
			rows := make([][]string, len(result.Data))
			for i, post := range result.Data {
				text := post.Text
				if len(text) > 50 {
					text = text[:47] + "..."
				}
				text = strings.ReplaceAll(text, "\n", " ")

				rows[i] = []string{
					post.ID,
					"@" + post.Username,
					text,
					post.MediaType,
					post.Timestamp.Format("2006-01-02"),
				}
			}

			return f.Table(headers, rows, []outfmt.ColumnType{
				outfmt.ColumnID,
				outfmt.ColumnPlain,
				outfmt.ColumnPlain,
				outfmt.ColumnStatus,
				outfmt.ColumnDate,
			})
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum results")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor")
	cmd.Flags().StringVar(&mediaType, "media-type", "", "Filter by media type (TEXT, IMAGE, VIDEO)")
	cmd.Flags().StringVar(&since, "since", "", "Posts after date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&until, "until", "", "Posts before date (YYYY-MM-DD)")

	return cmd
}
