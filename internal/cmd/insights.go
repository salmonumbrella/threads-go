package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	threads "github.com/salmonumbrella/threads-go"
	"github.com/salmonumbrella/threads-go/internal/outfmt"
	"github.com/salmonumbrella/threads-go/internal/ui"
)

var insightsCmd = &cobra.Command{
	Use:   "insights",
	Short: "Get insights and analytics",
	Long:  `Access insights and analytics data for posts and your account.`,
}

var insightsPostCmd = &cobra.Command{
	Use:   "post [post-id]",
	Short: "Get insights for a post",
	Long: `Get analytics insights for a specific post.

Available metrics: views, likes, replies, reposts, quotes, shares

Examples:
  threads insights post 12345678901234567
  threads insights post 12345678901234567 --metrics views,likes,replies
  threads insights post 12345678901234567 --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runInsightsPost,
}

var insightsAccountCmd = &cobra.Command{
	Use:   "account",
	Short: "Get account-level insights",
	Long: `Get analytics insights for your account.

Available metrics: views, likes, replies, reposts, quotes, clicks, followers_count, follower_demographics

Examples:
  threads insights account
  threads insights account --metrics views,followers_count
  threads insights account --period day
  threads insights account --output json`,
	RunE: runInsightsAccount,
}

// Insights command flags
var (
	insightsMetrics []string
	insightsPeriod  string
)

func init() {
	// Post insights flags
	insightsPostCmd.Flags().StringSliceVar(&insightsMetrics, "metrics", []string{"views", "likes", "replies", "reposts"}, "Metrics to retrieve (comma-separated)")

	// Account insights flags
	insightsAccountCmd.Flags().StringSliceVar(&insightsMetrics, "metrics", []string{"views", "likes", "replies", "reposts"}, "Metrics to retrieve (comma-separated)")
	insightsAccountCmd.Flags().StringVar(&insightsPeriod, "period", "lifetime", "Time period: day, lifetime")

	insightsCmd.AddCommand(insightsPostCmd)
	insightsCmd.AddCommand(insightsAccountCmd)
}

func runInsightsPost(cmd *cobra.Command, args []string) error {
	postID := args[0]

	ctx := cmd.Context()
	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	insights, err := client.GetPostInsights(ctx, threads.PostID(postID), insightsMetrics)
	if err != nil {
		return fmt.Errorf("failed to get post insights: %w", err)
	}

	format := outfmt.FromContext(ctx)

	if format == outfmt.JSON {
		return outfmt.WriteJSON(insights, jqQuery)
	}

	// Text output
	ui.Success("Post Insights for %s", postID)
	fmt.Println()

	if len(insights.Data) == 0 {
		ui.Info("No insights data available")
		return nil
	}

	f := outfmt.NewFormatter()
	f.Header("METRIC", "VALUE", "PERIOD")

	for _, insight := range insights.Data {
		value := 0
		if len(insight.Values) > 0 {
			value = insight.Values[0].Value
		} else if insight.TotalValue != nil {
			value = insight.TotalValue.Value
		}
		f.Row(insight.Name, value, insight.Period)
	}
	f.Flush()

	return nil
}

func runInsightsAccount(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	// Get the authenticated user's ID
	user, err := client.GetMe(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	insights, err := client.GetAccountInsights(ctx, threads.UserID(user.ID), insightsMetrics, insightsPeriod)
	if err != nil {
		return fmt.Errorf("failed to get account insights: %w", err)
	}

	format := outfmt.FromContext(ctx)

	if format == outfmt.JSON {
		return outfmt.WriteJSON(insights, jqQuery)
	}

	// Text output
	ui.Success("Account Insights for @%s", user.Username)
	fmt.Println()

	if len(insights.Data) == 0 {
		ui.Info("No insights data available")
		return nil
	}

	f := outfmt.NewFormatter()
	f.Header("METRIC", "VALUE", "PERIOD")

	for _, insight := range insights.Data {
		value := 0
		if len(insight.Values) > 0 {
			value = insight.Values[0].Value
		} else if insight.TotalValue != nil {
			value = insight.TotalValue.Value
		}
		f.Row(insight.Name, value, insight.Period)
	}
	f.Flush()

	return nil
}
