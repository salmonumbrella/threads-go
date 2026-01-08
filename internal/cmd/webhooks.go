package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	threads "github.com/salmonumbrella/threads-go"
	"github.com/salmonumbrella/threads-go/internal/iocontext"
	"github.com/salmonumbrella/threads-go/internal/outfmt"
)

// NewWebhooksCmd builds the webhooks command group.
func NewWebhooksCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhooks",
		Short: "Manage webhook subscriptions",
		Long: `Manage webhook subscriptions for receiving real-time notifications.

Webhooks allow you to receive instant notifications when events occur on Threads,
such as mentions, new posts, or deletions.

Supported events:
  - mentions:  Triggered when someone mentions you in a post
  - publishes: Triggered when you publish a new post
  - deletes:   Triggered when a post is deleted

Your callback URL must be:
  - HTTPS (required by Meta's API)
  - Publicly accessible
  - Able to respond to verification challenges`,
	}

	cmd.AddCommand(newWebhooksSubscribeCmd(f))
	cmd.AddCommand(newWebhooksListCmd(f))
	cmd.AddCommand(newWebhooksDeleteCmd(f))

	return cmd
}

func newWebhooksSubscribeCmd(f *Factory) *cobra.Command {
	var (
		callbackURL string
		verifyToken string
		events      []string
	)

	cmd := &cobra.Command{
		Use:   "subscribe",
		Short: "Subscribe to webhook events",
		Long: `Create a new webhook subscription to receive real-time notifications.

Your callback URL must be HTTPS and publicly accessible. Meta will send a
verification request to your endpoint during subscription setup.

Supported events:
  - mentions:  Triggered when someone mentions you in a post
  - publishes: Triggered when you publish a new post
  - deletes:   Triggered when a post is deleted`,
		Example: `  # Subscribe to mention events
  threads webhooks subscribe --event mentions --url https://example.com/webhooks

  # Subscribe to multiple events
  threads webhooks subscribe --event mentions --event publishes --url https://example.com/webhooks

  # Subscribe with a verify token
  threads webhooks subscribe --event mentions --url https://example.com/webhooks --verify-token my-secret`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if callbackURL == "" {
				return &UserFriendlyError{
					Message:    "Callback URL is required",
					Suggestion: "Provide a callback URL with --url https://example.com/webhooks",
				}
			}

			if err := ValidateHTTPSURL(callbackURL, "Callback URL"); err != nil {
				return err
			}

			if len(events) == 0 {
				return &UserFriendlyError{
					Message:    "At least one event type is required",
					Suggestion: "Specify events with --event. Valid events: mentions, publishes, deletes",
				}
			}

			var webhookEvents []threads.WebhookEventType
			for _, event := range events {
				switch strings.ToLower(event) {
				case "mentions":
					webhookEvents = append(webhookEvents, threads.WebhookEventMentions)
				case "publishes":
					webhookEvents = append(webhookEvents, threads.WebhookEventPublishes)
				case "deletes":
					webhookEvents = append(webhookEvents, threads.WebhookEventDeletes)
				default:
					return &UserFriendlyError{
						Message:    fmt.Sprintf("Invalid event type: %s", event),
						Suggestion: "Valid event types are: mentions, publishes, deletes",
					}
				}
			}

			client, err := f.Client(ctx)
			if err != nil {
				return err
			}

			opts := &threads.WebhookSubscribeOptions{
				CallbackURL: callbackURL,
				VerifyToken: verifyToken,
				Fields:      webhookEvents,
			}

			subscription, err := client.SubscribeWebhook(ctx, opts)
			if err != nil {
				return WrapError("failed to create webhook subscription", err)
			}

			io := iocontext.GetIO(ctx)
			if outfmt.IsJSON(ctx) {
				return outfmt.WriteJSONTo(io.Out, webhookSubscriptionToMap(subscription), outfmt.GetQuery(ctx))
			}

			f.UI(ctx).Success("Webhook subscription created successfully!")
			fmt.Fprintf(io.Out, "  Callback URL: %s\n", subscription.CallbackURL)                 //nolint:errcheck // Best-effort output
			fmt.Fprintf(io.Out, "  Events:       %s\n", formatWebhookFields(subscription.Fields)) //nolint:errcheck // Best-effort output
			fmt.Fprintf(io.Out, "  Active:       %v\n", subscription.Active)                      //nolint:errcheck // Best-effort output

			return nil
		},
	}

	cmd.Flags().StringVar(&callbackURL, "url", "", "HTTPS callback URL to receive webhook events (required)")
	cmd.Flags().StringSliceVar(&events, "event", nil, "Event types to subscribe to: mentions, publishes, deletes (can be specified multiple times)")
	cmd.Flags().StringVar(&verifyToken, "verify-token", "", "Token to verify webhook callbacks (optional but recommended)")

	//nolint:errcheck,gosec // MarkFlagRequired cannot fail for flags that exist
	cmd.MarkFlagRequired("url")
	//nolint:errcheck,gosec // MarkFlagRequired cannot fail for flags that exist
	cmd.MarkFlagRequired("event")

	return cmd
}

func newWebhooksListCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List active webhook subscriptions",
		Long:  `List all active webhook subscriptions for your Threads app.`,
		Example: `  # List all subscriptions
  threads webhooks list

  # Output as JSON
  threads webhooks list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			client, err := f.Client(ctx)
			if err != nil {
				return err
			}

			result, err := client.ListWebhookSubscriptions(ctx)
			if err != nil {
				return WrapError("failed to list webhook subscriptions", err)
			}

			io := iocontext.GetIO(ctx)
			if outfmt.IsJSON(ctx) {
				return outfmt.WriteJSONTo(io.Out, result, outfmt.GetQuery(ctx))
			}

			out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
			if len(result.Data) == 0 {
				out.Empty("No webhook subscriptions found")
				return nil
			}

			headers := []string{"OBJECT", "CALLBACK URL", "FIELDS", "ACTIVE"}
			rows := make([][]string, len(result.Data))

			for i, sub := range result.Data {
				active := "no"
				if sub.Active {
					active = "yes"
				}
				rows[i] = []string{
					sub.Object,
					truncateURL(sub.CallbackURL, 40),
					formatWebhookFields(sub.Fields),
					active,
				}
			}

			return out.Table(headers, rows, []outfmt.ColumnType{
				outfmt.ColumnPlain,
				outfmt.ColumnPlain,
				outfmt.ColumnPlain,
				outfmt.ColumnStatus,
			})
		},
	}

	return cmd
}

func newWebhooksDeleteCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [subscription-id]",
		Short: "Delete a webhook subscription",
		Long: `Delete a webhook subscription by its ID or object type.

After deletion, your callback URL will no longer receive events for this subscription.`,
		Example: `  # Delete a subscription
  threads webhooks delete user

  # Delete with confirmation skip
  threads webhooks delete user --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			subscriptionID := args[0]

			io := iocontext.GetIO(ctx)
			if !outfmt.GetYes(ctx) {
				fmt.Fprintf(io.Out, "Webhook subscription to delete: %s\n\n", subscriptionID) //nolint:errcheck // Best-effort output
				if !f.Confirm(ctx, "Delete this webhook subscription?") {
					fmt.Fprintln(io.Out, "Cancelled.") //nolint:errcheck // Best-effort output
					return nil
				}
			}

			client, err := f.Client(ctx)
			if err != nil {
				return err
			}

			if err := client.DeleteWebhookSubscription(ctx, subscriptionID); err != nil {
				return WrapError("failed to delete webhook subscription", err)
			}

			if outfmt.IsJSON(ctx) {
				return outfmt.WriteJSONTo(io.Out, map[string]any{
					"success": true,
					"deleted": subscriptionID,
				}, outfmt.GetQuery(ctx))
			}

			f.UI(ctx).Success("Webhook subscription deleted successfully")
			return nil
		},
	}

	return cmd
}

// webhookSubscriptionToMap converts a WebhookSubscription to a map for JSON output
func webhookSubscriptionToMap(sub *threads.WebhookSubscription) map[string]any {
	fields := make([]string, len(sub.Fields))
	for i, f := range sub.Fields {
		fields[i] = f.Name
	}

	return map[string]any{
		"id":           sub.ID,
		"object":       sub.Object,
		"callback_url": sub.CallbackURL,
		"fields":       fields,
		"active":       sub.Active,
		"created_time": sub.CreatedTime,
	}
}

// formatWebhookFields formats webhook fields for display
func formatWebhookFields(fields []threads.WebhookField) string {
	if len(fields) == 0 {
		return "-"
	}

	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}
	return strings.Join(names, ", ")
}

// truncateURL truncates a URL for display
func truncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen-3] + "..."
}
