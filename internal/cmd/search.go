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
		all        bool
		noHints    bool
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

			if best && all {
				return &UserFriendlyError{
					Message:    "Cannot combine --best and --all",
					Suggestion: "Use --best for a single result, or --all to paginate all results",
				}
			}

			if best {
				if emit == "" {
					emit = "json"
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
				emMode, errMode := parseEmitMode(emit)
				if errMode != nil {
					return errMode
				}
				return emitResult(ctx, io, emMode, item.ID, item.Permalink, item)
			}

			if all {
				out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
				var allPosts []api.Post
				var allRows [][]string
				var lastPaging api.Paging

				// Use the already-fetched result as the first page, then paginate from its cursor.
				page := result
				for {
					lastPaging = page.Paging
					next := pagingAfter(page.Paging)

					if outfmt.IsJSONL(ctx) {
						if errOut := out.Output(page.Data); errOut != nil {
							return errOut
						}
					} else if outfmt.GetFormat(ctx) == outfmt.JSON {
						allPosts = append(allPosts, page.Data...)
					} else {
						for _, post := range page.Data {
							text := post.Text
							if len(text) > 50 {
								text = text[:47] + "..."
							}
							text = strings.ReplaceAll(text, "\n", " ")
							allRows = append(allRows, []string{
								post.ID,
								"@" + post.Username,
								text,
								post.MediaType,
								post.Timestamp.Format("2006-01-02"),
							})
						}
					}

					if next == "" || len(page.Data) == 0 {
						break
					}
					opts.After = next
					nextPage, errPage := client.KeywordSearch(ctx, query, opts)
					if errPage != nil {
						return WrapError("search failed", errPage)
					}
					page = nextPage
				}

				if outfmt.GetFormat(ctx) == outfmt.JSON {
					items := allPosts
					if len(items) == 0 {
						items = []api.Post{}
					}
					// After fetching all pages, cursor is empty and has_more=false.
					return out.Output(itemsEnvelope(items, lastPaging, ""))
				}
				if outfmt.GetFormat(ctx) == outfmt.Text {
					if len(allRows) == 0 {
						out.Empty("No results found")
						return nil
					}
					return out.Table([]string{"ID", "USER", "TEXT", "TYPE", "DATE"}, allRows, []outfmt.ColumnType{
						outfmt.ColumnID,
						outfmt.ColumnPlain,
						outfmt.ColumnPlain,
						outfmt.ColumnStatus,
						outfmt.ColumnDate,
					})
				}
				return nil
			}

			next := pagingAfter(result.Paging)
			if !noHints && next != "" && io.ErrOut != nil && (outfmt.IsJSONL(ctx) || outfmt.GetFormat(ctx) == outfmt.Text) {
				fmt.Fprintf(io.ErrOut, "\nMore results available. Use --cursor %s to see next page.\n", next) //nolint:errcheck // Best-effort output
			}

			if outfmt.IsJSONL(ctx) {
				out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
				return out.Output(result.Data)
			}
			if outfmt.GetFormat(ctx) == outfmt.JSON {
				out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
				items := result.Data
				if len(items) == 0 {
					items = []api.Post{}
				}
				return out.Output(itemsEnvelope(items, result.Paging, next))
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
	cmd.Flags().BoolVar(&all, "all", false, "Fetch all pages (auto-paginate)")
	cmd.Flags().BoolVar(&noHints, "no-hints", false, "Suppress pagination hints on stderr")

	return cmd
}
