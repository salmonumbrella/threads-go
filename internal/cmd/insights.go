package cmd

import (
	"fmt"
	"strings"

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

Available metrics: views, likes, replies, reposts, quotes, shares, link_clicks, profile_clicks

Click metrics:
  link_clicks    - Number of clicks on links in the post
  profile_clicks - Number of clicks to view your profile from the post

Examples:
  threads insights post 12345678901234567
  threads insights post 12345678901234567 --metrics views,likes,replies
  threads insights post 12345678901234567 --metrics link_clicks,profile_clicks
  threads insights post 12345678901234567 --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runInsightsPost,
}

var insightsAccountCmd = &cobra.Command{
	Use:   "account",
	Short: "Get account-level insights",
	Long: `Get analytics insights for your account.

Available metrics: views, likes, replies, reposts, quotes, clicks, followers_count, follower_demographics

Metric details:
  clicks - Total clicks across all posts (combined link and profile clicks)

Breakdown options (for follower_demographics metric):
  country - Breakdown by country
  city    - Breakdown by city
  age     - Breakdown by age group
  gender  - Breakdown by gender

Examples:
  threads insights account
  threads insights account --metrics views,followers_count
  threads insights account --metrics clicks
  threads insights account --period day
  threads insights account --metrics follower_demographics --breakdown country
  threads insights account --metrics follower_demographics --breakdown age
  threads insights account --output json`,
	RunE: runInsightsAccount,
}

// Insights command flags
var (
	insightsMetrics   []string
	insightsPeriod    string
	insightsBreakdown string
)

func init() {
	// Post insights flags
	insightsPostCmd.Flags().StringSliceVar(&insightsMetrics, "metrics", []string{"views", "likes", "replies", "reposts"}, "Metrics to retrieve (comma-separated)")

	// Account insights flags
	insightsAccountCmd.Flags().StringSliceVar(&insightsMetrics, "metrics", []string{"views", "likes", "replies", "reposts"}, "Metrics to retrieve (comma-separated)")
	insightsAccountCmd.Flags().StringVar(&insightsPeriod, "period", "lifetime", "Time period: day, lifetime")
	insightsAccountCmd.Flags().StringVar(&insightsBreakdown, "breakdown", "", "Breakdown for follower_demographics: country, city, age, gender")

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
		return WrapError("failed to get post insights", err)
	}

	if outfmt.IsJSON(ctx) {
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

	// Validate breakdown if provided
	if insightsBreakdown != "" {
		validBreakdowns := map[string]bool{
			"country": true,
			"city":    true,
			"age":     true,
			"gender":  true,
		}
		if !validBreakdowns[insightsBreakdown] {
			return &UserFriendlyError{
				Message:    fmt.Sprintf("Invalid breakdown value: %s", insightsBreakdown),
				Suggestion: "Valid breakdown values are: country, city, age, gender",
			}
		}
	}

	// Get the authenticated user's ID
	user, err := client.GetMe(ctx)
	if err != nil {
		return WrapError("failed to get user info", err)
	}

	// Build options
	opts := &threads.AccountInsightsOptions{
		Breakdown: insightsBreakdown,
	}

	// Convert string metrics to AccountInsightMetric
	for _, m := range insightsMetrics {
		opts.Metrics = append(opts.Metrics, threads.AccountInsightMetric(m))
	}

	// Set period
	if insightsPeriod != "" {
		opts.Period = threads.InsightPeriod(insightsPeriod)
	}

	insights, err := client.GetAccountInsightsWithOptions(ctx, threads.UserID(user.ID), opts)
	if err != nil {
		return WrapError("failed to get account insights", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(insights, jqQuery)
	}

	// Text output
	ui.Success("Account Insights for @%s", user.Username)
	fmt.Println()

	if len(insights.Data) == 0 {
		ui.Info("No insights data available")
		return nil
	}

	// Check if this is a breakdown result (follower_demographics with breakdown)
	hasBreakdownData := false
	for _, insight := range insights.Data {
		if insight.Name == "follower_demographics" && len(insight.Values) > 0 {
			hasBreakdownData = true
			break
		}
	}

	if hasBreakdownData && insightsBreakdown != "" {
		// Display breakdown data in a special format
		f := outfmt.NewFormatter()
		f.Header(strings.ToUpper(insightsBreakdown), "PERCENTAGE")

		for _, insight := range insights.Data {
			if insight.Name == "follower_demographics" {
				for _, v := range insight.Values {
					// Values for breakdown contain the category and percentage
					f.Row(v.EndTime, fmt.Sprintf("%d%%", v.Value))
				}
			}
		}
		f.Flush()
	} else {
		// Standard metrics display
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
	}

	return nil
}
