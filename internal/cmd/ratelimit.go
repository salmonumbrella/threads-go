package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

// NewRateLimitCmd builds the ratelimit command group.
func NewRateLimitCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ratelimit",
		Aliases: []string{"rate", "limits"},
		Short:   "View rate limit status",
	}

	cmd.AddCommand(newRateLimitStatusCmd(f))
	cmd.AddCommand(newRateLimitPublishingCmd(f))

	return cmd
}

func newRateLimitStatusCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current rate limit status",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := f.Client(ctx)
			if err != nil {
				return err
			}

			status := client.GetRateLimitStatus()
			isLimited := client.IsRateLimited()
			nearLimit := client.IsNearRateLimit(0.8)

			io := iocontext.GetIO(ctx)
			if outfmt.IsJSON(ctx) {
				out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
				return out.Output(map[string]interface{}{
					"is_limited": isLimited,
					"remaining":  status.Remaining,
					"limit":      status.Limit,
					"reset_at":   status.ResetTime,
					"reset_in":   status.ResetIn.String(),
					"near_limit": nearLimit,
				})
			}

			// Text output
			if isLimited {
				fmt.Fprintf(io.Out, "Rate limited until %s\n", status.ResetTime.Format(time.RFC3339)) //nolint:errcheck // Best-effort output
			} else {
				fmt.Fprintf(io.Out, "Remaining: %d/%d\n", status.Remaining, status.Limit) //nolint:errcheck // Best-effort output
				if nearLimit {
					fmt.Fprintln(io.Out, "Warning: Near rate limit threshold") //nolint:errcheck // Best-effort output
				}
			}

			return nil
		},
	}
	return cmd
}

func newRateLimitPublishingCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publishing",
		Short: "Show publishing limits (API quota)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := f.Client(ctx)
			if err != nil {
				return err
			}

			limits, err := client.GetPublishingLimits(ctx)
			if err != nil {
				return WrapError("failed to get publishing limits", err)
			}

			io := iocontext.GetIO(ctx)
			if outfmt.IsJSON(ctx) {
				out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
				return out.Output(limits)
			}

			// Text output
			fmt.Fprintf(io.Out, "%v\n", limits) //nolint:errcheck // Best-effort output
			return nil
		},
	}
	return cmd
}
