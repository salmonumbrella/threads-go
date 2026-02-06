package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/threads-cli/internal/api"
	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

// NewInsightsCmd builds the insights command group.
func NewInsightsCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "insights",
		Short: "Get insights and analytics",
		Long:  `Access insights and analytics data for posts and your account.`,
	}

	cmd.AddCommand(newInsightsPostCmd(f))
	cmd.AddCommand(newInsightsAccountCmd(f))

	return cmd
}

type insightsPostOptions struct {
	Metrics []string
}

func newInsightsPostCmd(f *Factory) *cobra.Command {
	opts := &insightsPostOptions{
		Metrics: []string{"views", "likes", "replies", "reposts"},
	}

	cmd := &cobra.Command{
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInsightsPost(cmd, f, opts, args[0])
		},
	}

	cmd.Flags().StringSliceVar(&opts.Metrics, "metrics", opts.Metrics, "Metrics to retrieve (comma-separated)")
	return cmd
}

func runInsightsPost(cmd *cobra.Command, f *Factory, opts *insightsPostOptions, postID string) error {
	ctx := cmd.Context()
	client, err := f.Client(ctx)
	if err != nil {
		return err
	}

	insights, err := client.GetPostInsights(ctx, api.PostID(postID), opts.Metrics)
	if err != nil {
		return WrapError("failed to get post insights", err)
	}

	io := iocontext.GetIO(ctx)
	if outfmt.IsJSON(ctx) {
		out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
		return out.Output(insights)
	}

	p := f.UI(ctx)
	p.Success("Post Insights for %s", postID)
	fmt.Fprintln(io.Out) //nolint:errcheck // Best-effort output

	if len(insights.Data) == 0 {
		p.Info("No insights data available")
		return nil
	}

	fmtr := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
	fmtr.Header("METRIC", "VALUE", "PERIOD")

	for _, insight := range insights.Data {
		value := 0
		if len(insight.Values) > 0 {
			value = insight.Values[0].Value
		} else if insight.TotalValue != nil {
			value = insight.TotalValue.Value
		}
		fmtr.Row(insight.Name, value, insight.Period)
	}
	fmtr.Flush()

	return nil
}

type insightsAccountOptions struct {
	Metrics   []string
	Period    string
	Breakdown string
}

func newInsightsAccountCmd(f *Factory) *cobra.Command {
	opts := &insightsAccountOptions{
		Metrics: []string{"views", "likes", "replies", "reposts"},
		Period:  "lifetime",
	}

	cmd := &cobra.Command{
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInsightsAccount(cmd, f, opts)
		},
	}

	cmd.Flags().StringSliceVar(&opts.Metrics, "metrics", opts.Metrics, "Metrics to retrieve (comma-separated)")
	cmd.Flags().StringVar(&opts.Period, "period", opts.Period, "Time period: day, lifetime")
	cmd.Flags().StringVar(&opts.Breakdown, "breakdown", "", "Breakdown for follower_demographics: country, city, age, gender")

	return cmd
}

func runInsightsAccount(cmd *cobra.Command, f *Factory, opts *insightsAccountOptions) error {
	ctx := cmd.Context()
	client, err := f.Client(ctx)
	if err != nil {
		return err
	}

	if opts.Breakdown != "" {
		validBreakdowns := map[string]bool{
			"country": true,
			"city":    true,
			"age":     true,
			"gender":  true,
		}
		if !validBreakdowns[opts.Breakdown] {
			return &UserFriendlyError{
				Message:    fmt.Sprintf("Invalid breakdown value: %s", opts.Breakdown),
				Suggestion: "Valid breakdown values are: country, city, age, gender",
			}
		}
	}

	creds, err := f.ActiveCredentials(ctx)
	if err != nil {
		return err
	}

	optsReq := &api.AccountInsightsOptions{
		Breakdown: opts.Breakdown,
	}

	for _, m := range opts.Metrics {
		optsReq.Metrics = append(optsReq.Metrics, api.AccountInsightMetric(m))
	}

	if opts.Period != "" {
		optsReq.Period = api.InsightPeriod(opts.Period)
	}

	insights, err := client.GetAccountInsightsWithOptions(ctx, api.UserID(creds.UserID), optsReq)
	if err != nil {
		return WrapError("failed to get account insights", err)
	}

	io := iocontext.GetIO(ctx)
	if outfmt.IsJSON(ctx) {
		out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
		return out.Output(insights)
	}

	p := f.UI(ctx)
	p.Success("Account Insights for @%s", creds.Username)
	fmt.Fprintln(io.Out) //nolint:errcheck // Best-effort output

	if len(insights.Data) == 0 {
		p.Info("No insights data available")
		return nil
	}

	hasBreakdownData := false
	for _, insight := range insights.Data {
		if insight.Name == "follower_demographics" && len(insight.Values) > 0 {
			hasBreakdownData = true
			break
		}
	}

	if hasBreakdownData && opts.Breakdown != "" {
		fmtr := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
		fmtr.Header(strings.ToUpper(opts.Breakdown), "PERCENTAGE")

		for _, insight := range insights.Data {
			if insight.Name == "follower_demographics" {
				for _, v := range insight.Values {
					fmtr.Row(v.EndTime, fmt.Sprintf("%d%%", v.Value))
				}
			}
		}
		fmtr.Flush()
		return nil
	}

	fmtr := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
	fmtr.Header("METRIC", "VALUE", "PERIOD")

	for _, insight := range insights.Data {
		value := 0
		if len(insight.Values) > 0 {
			value = insight.Values[0].Value
		} else if insight.TotalValue != nil {
			value = insight.TotalValue.Value
		}
		fmtr.Row(insight.Name, value, insight.Period)
	}
	fmtr.Flush()

	return nil
}
