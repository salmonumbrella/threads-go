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
	var mode string
	var searchType string

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search posts by keyword or topic tag",
		Long: `Search posts by keyword or topic tag.

By default, searches for keywords. Use --mode=tag to search for topic tags instead.
Results can be sorted by popularity (top) or recency (recent).`,
		Example: `  # Search for keyword
  threads search "coffee"

  # Search for topic tag
  threads search "coffee" --mode=tag

  # Get most recent results
  threads search "coffee" --type=recent

  # Combine options
  threads search "technology" --mode=tag --type=recent --media-type=IMAGE`,
		Args: cobra.ExactArgs(1),
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

			// Handle search mode
			switch strings.ToLower(mode) {
			case "keyword", "":
				opts.SearchMode = threads.SearchModeKeyword
			case "tag":
				opts.SearchMode = threads.SearchModeTag
			default:
				return &UserFriendlyError{
					Message:    fmt.Sprintf("Invalid --mode value: %s", mode),
					Suggestion: "Use 'keyword' (default) or 'tag'",
				}
			}

			// Handle search type
			switch strings.ToLower(searchType) {
			case "top", "":
				opts.SearchType = threads.SearchTypeTop
			case "recent":
				opts.SearchType = threads.SearchTypeRecent
			default:
				return &UserFriendlyError{
					Message:    fmt.Sprintf("Invalid --type value: %s", searchType),
					Suggestion: "Use 'top' (default) or 'recent'",
				}
			}

			if mediaType != "" {
				opts.MediaType = mediaType
			}

			if since != "" {
				sinceTime, errSince := time.Parse("2006-01-02", since)
				if errSince != nil {
					return &UserFriendlyError{
						Message:    fmt.Sprintf("Invalid --since date: %s", since),
						Suggestion: "Use YYYY-MM-DD format (e.g., 2024-01-15)",
					}
				}
				opts.Since = sinceTime.Unix()
			}

			if until != "" {
				untilTime, errUntil := time.Parse("2006-01-02", until)
				if errUntil != nil {
					return &UserFriendlyError{
						Message:    fmt.Sprintf("Invalid --until date: %s", until),
						Suggestion: "Use YYYY-MM-DD format (e.g., 2024-01-15)",
					}
				}
				opts.Until = untilTime.Unix()
			}

			result, err := client.KeywordSearch(ctx, query, opts)
			if err != nil {
				return WrapError("search failed", err)
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
	cmd.Flags().StringVar(&mode, "mode", "keyword", "Search mode: keyword (default) or tag")
	cmd.Flags().StringVar(&searchType, "type", "top", "Result type: top (default) or recent")

	return cmd
}
