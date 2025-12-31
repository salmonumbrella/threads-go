package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	threads "github.com/salmonumbrella/threads-go"
	"github.com/salmonumbrella/threads-go/internal/outfmt"
	"github.com/salmonumbrella/threads-go/internal/ui"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search posts by keyword",
	Long: `Search for posts on Threads by keyword.

Examples:
  threads search "machine learning"
  threads search golang --limit 10
  threads search "AI news" --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func init() {
	// The --limit flag is already defined globally in root.go
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	ctx := cmd.Context()
	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	opts := &threads.SearchOptions{}
	if limitFlag > 0 {
		opts.Limit = limitFlag
	} else {
		opts.Limit = 25 // Default limit
	}

	results, err := client.KeywordSearch(ctx, query, opts)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(results, jqQuery)
	}

	// Text output
	if len(results.Data) == 0 {
		ui.Info("No results found for %q", query)
		return nil
	}

	ui.Success("Found %d posts matching %q", len(results.Data), query)
	fmt.Println()

	for _, post := range results.Data {
		// Format post ID
		fmt.Printf("%s  ", ui.Bold(post.ID))

		// Username
		fmt.Printf("@%s", post.Username)

		// Timestamp
		if !post.Timestamp.IsZero() {
			fmt.Printf("  %s", ui.Dim(post.Timestamp.Format("Jan 2, 2006")))
		}
		fmt.Println()

		// Post text (truncated)
		if post.Text != "" {
			text := post.Text
			// Truncate long text
			if len(text) > 100 {
				text = text[:100] + "..."
			}
			// Remove newlines for cleaner display
			text = strings.ReplaceAll(text, "\n", " ")
			fmt.Printf("  %s\n", text)
		}

		// Media type indicator
		if post.MediaType != "" && post.MediaType != "TEXT" {
			fmt.Printf("  [%s]\n", post.MediaType)
		}

		fmt.Println()
	}

	return nil
}
