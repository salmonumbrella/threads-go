package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/threads-cli/internal/api"
	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

// NewSearchCmd builds the search command.
func NewSearchCmd(f *Factory) *cobra.Command {
	var (
		limit      int
		cursor     string
		mediaType  string
		since      string
		until      string
		mode       string
		searchType string
		best       bool
		emit       string
	)

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

			if best {
				emit = strings.ToLower(strings.TrimSpace(emit))
				if emit == "" {
					emit = "json"
				}
				switch emit {
				case "json", "id", "url":
				default:
					return &UserFriendlyError{
						Message:    fmt.Sprintf("Invalid --emit value: %s", emit),
						Suggestion: "Valid values are: json, id, url",
					}
				}
			}

			client, err := f.Client(ctx)
			if err != nil {
				return err
			}

			opts := &api.SearchOptions{
				Limit: limit,
				After: cursor,
			}

			// Handle search mode
			switch strings.ToLower(mode) {
			case "keyword", "":
				opts.SearchMode = api.SearchModeKeyword
			case "tag":
				opts.SearchMode = api.SearchModeTag
			default:
				return &UserFriendlyError{
					Message:    fmt.Sprintf("Invalid --mode value: %s", mode),
					Suggestion: "Use 'keyword' (default) or 'tag'",
				}
			}

			// Handle search type
			switch strings.ToLower(searchType) {
			case "top", "":
				opts.SearchType = api.SearchTypeTop
			case "recent":
				opts.SearchType = api.SearchTypeRecent
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

			io := iocontext.GetIO(ctx)

			if best {
				if len(result.Data) == 0 {
					return &UserFriendlyError{
						Message:    "No results found",
						Suggestion: "Try a different query or broaden your search",
					}
				}

				item := result.Data[0]

				// When best+emit is requested, allow emitting a scalar in text mode for easy chaining.
				if !outfmt.IsJSON(ctx) {
					switch emit {
					case "id":
						fmt.Fprintln(io.Out, item.ID) //nolint:errcheck // Best-effort output
						return nil
					case "url":
						if strings.TrimSpace(item.Permalink) == "" {
							return &UserFriendlyError{
								Message:    "Cannot emit url: permalink is empty",
								Suggestion: "Use --emit id or --emit json",
							}
						}
						fmt.Fprintln(io.Out, item.Permalink) //nolint:errcheck // Best-effort output
						return nil
					}
				}

				// JSON mode: emit stable wrapper.
				if outfmt.IsJSON(ctx) {
					switch emit {
					case "id":
						return outfmt.WriteJSONTo(io.Out, map[string]any{"id": item.ID}, outfmt.GetQuery(ctx))
					case "url":
						return outfmt.WriteJSONTo(io.Out, map[string]any{"url": item.Permalink}, outfmt.GetQuery(ctx))
					default:
						return outfmt.WriteJSONTo(io.Out, map[string]any{
							"id":   item.ID,
							"item": item,
						}, outfmt.GetQuery(ctx))
					}
				}

				// Text mode default.
				fmt.Fprintf(io.Out, "%s\n", item.ID) //nolint:errcheck // Best-effort output
				if strings.TrimSpace(item.Permalink) != "" {
					fmt.Fprintf(io.Out, "%s\n", item.Permalink) //nolint:errcheck // Best-effort output
				}
				return nil
			}

			if outfmt.IsJSON(ctx) {
				return outfmt.WriteJSONTo(io.Out, result, outfmt.GetQuery(ctx))
			}

			out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))

			if len(result.Data) == 0 {
				out.Empty("No results found")
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

			return out.Table(headers, rows, []outfmt.ColumnType{
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
	cmd.Flags().BoolVar(&best, "best", false, "Auto-select the best result (non-interactive)")
	cmd.Flags().StringVar(&emit, "emit", "json", "When using --best, emit: json|id|url")

	return cmd
}
