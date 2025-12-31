package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/threads-go/internal/outfmt"
)

func newRateLimitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ratelimit",
		Aliases: []string{"rate", "limits"},
		Short:   "View rate limit status",
	}

	cmd.AddCommand(newRateLimitStatusCmd())
	cmd.AddCommand(newRateLimitPublishingCmd())

	return cmd
}

func newRateLimitStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current rate limit status",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd.Context())
			if err != nil {
				return err
			}

			status := client.GetRateLimitStatus()
			isLimited := client.IsRateLimited()
			nearLimit := client.IsNearRateLimit(0.8)

			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(map[string]interface{}{
					"is_limited": isLimited,
					"remaining":  status.Remaining,
					"limit":      status.Limit,
					"reset_at":   status.ResetTime,
					"reset_in":   status.ResetIn.String(),
					"near_limit": nearLimit,
				}, jqQuery)
			}

			// Text output
			if isLimited {
				fmt.Printf("Rate limited until %s\n", status.ResetTime.Format(time.RFC3339))
			} else {
				fmt.Printf("Remaining: %d/%d\n", status.Remaining, status.Limit)
				if nearLimit {
					fmt.Println("Warning: Near rate limit threshold")
				}
			}

			return nil
		},
	}
	return cmd
}

func newRateLimitPublishingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publishing",
		Short: "Show publishing limits (API quota)",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient(cmd.Context())
			if err != nil {
				return err
			}

			limits, err := client.GetPublishingLimits(cmd.Context())
			if err != nil {
				return WrapError("failed to get publishing limits", err)
			}

			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(limits, jqQuery)
			}

			// Text output
			fmt.Printf("%v\n", limits)
			return nil
		},
	}
	return cmd
}
