package threads

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// WebhookEventType represents the types of events that can trigger webhooks
type WebhookEventType string

const (
	// WebhookEventMentions triggers when someone mentions you in a post
	WebhookEventMentions WebhookEventType = "mentions"
	// WebhookEventPublishes triggers when you publish a post
	WebhookEventPublishes WebhookEventType = "publishes"
	// WebhookEventDeletes triggers when a post is deleted
	WebhookEventDeletes WebhookEventType = "deletes"
)

// WebhookSubscription represents a webhook subscription configuration
type WebhookSubscription struct {
	ID          string         `json:"id"`
	Object      string         `json:"object"`
	CallbackURL string         `json:"callback_url"`
	Fields      []WebhookField `json:"fields,omitempty"`
	Active      bool           `json:"active"`
	CreatedTime string         `json:"created_time,omitempty"`
}

// WebhookField represents a field in a webhook subscription
type WebhookField struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// WebhookSubscriptionsResponse represents the response from listing webhook subscriptions
type WebhookSubscriptionsResponse struct {
	Data   []WebhookSubscription `json:"data"`
	Paging *Paging               `json:"paging,omitempty"`
}

// WebhookSubscribeOptions contains options for subscribing to webhooks
type WebhookSubscribeOptions struct {
	// CallbackURL is the HTTPS URL that will receive webhook callbacks
	CallbackURL string
	// VerifyToken is a token used to verify the webhook subscription
	VerifyToken string
	// Fields specifies which events to subscribe to (e.g., "mentions", "publishes", "deletes")
	Fields []WebhookEventType
}

// WebhookManager provides methods for managing webhook subscriptions
type WebhookManager interface {
	// SubscribeWebhook creates a new webhook subscription
	SubscribeWebhook(ctx context.Context, opts *WebhookSubscribeOptions) (*WebhookSubscription, error)

	// ListWebhookSubscriptions lists all active webhook subscriptions
	ListWebhookSubscriptions(ctx context.Context) (*WebhookSubscriptionsResponse, error)

	// DeleteWebhookSubscription deletes a webhook subscription by ID
	DeleteWebhookSubscription(ctx context.Context, subscriptionID string) error
}

// SubscribeWebhook creates a new webhook subscription for the authenticated user's app.
// The callbackURL must be HTTPS and publicly accessible.
// The verifyToken is sent by Meta to verify your endpoint during subscription setup.
func (c *Client) SubscribeWebhook(ctx context.Context, opts *WebhookSubscribeOptions) (*WebhookSubscription, error) {
	if opts == nil {
		return nil, NewValidationError(400, "Options required", "WebhookSubscribeOptions cannot be nil", "opts")
	}

	if opts.CallbackURL == "" {
		return nil, NewValidationError(400, "Callback URL required", "CallbackURL is required for webhook subscription", "callback_url")
	}

	if len(opts.Fields) == 0 {
		return nil, NewValidationError(400, "Fields required", "At least one event field is required", "fields")
	}

	// Build fields string (comma-separated)
	var fields string
	for i, field := range opts.Fields {
		if i > 0 {
			fields += ","
		}
		fields += string(field)
	}

	// Get the app ID from the config (client ID is the app ID in Meta's API)
	appID := c.config.ClientID

	// Build form data for the POST request
	formData := url.Values{}
	formData.Set("object", "user")
	formData.Set("callback_url", opts.CallbackURL)
	formData.Set("fields", fields)
	formData.Set("access_token", c.accessToken)
	if opts.VerifyToken != "" {
		formData.Set("verify_token", opts.VerifyToken)
	}

	c.mu.RLock()
	token := c.accessToken
	c.mu.RUnlock()

	// POST to /{app-id}/subscriptions
	resp, err := c.httpClient.POST(
		fmt.Sprintf("/v1.0/%s/subscriptions", appID),
		formData,
		token,
	)
	if err != nil {
		return nil, err
	}

	// Parse the response
	var result struct {
		Success bool `json:"success"`
	}
	if err := safeJSONUnmarshal(resp.Body, &result, "subscribe webhook", resp.RequestID); err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, NewAPIError(resp.StatusCode, "Failed to create webhook subscription", string(resp.Body), resp.RequestID)
	}

	// Return a subscription object (the actual ID may need to be fetched from list)
	subscription := &WebhookSubscription{
		Object:      "user",
		CallbackURL: opts.CallbackURL,
		Active:      true,
	}

	// Populate fields
	for _, field := range opts.Fields {
		subscription.Fields = append(subscription.Fields, WebhookField{Name: string(field)})
	}

	return subscription, nil
}

// ListWebhookSubscriptions retrieves all webhook subscriptions for the authenticated user's app.
func (c *Client) ListWebhookSubscriptions(ctx context.Context) (*WebhookSubscriptionsResponse, error) {
	// Get the app ID from the config
	appID := c.config.ClientID

	c.mu.RLock()
	token := c.accessToken
	c.mu.RUnlock()

	params := url.Values{}
	params.Set("access_token", token)

	// GET /{app-id}/subscriptions
	resp, err := c.httpClient.GET(
		fmt.Sprintf("/v1.0/%s/subscriptions", appID),
		params,
		token,
	)
	if err != nil {
		return nil, err
	}

	var result WebhookSubscriptionsResponse
	if err := safeJSONUnmarshal(resp.Body, &result, "list webhook subscriptions", resp.RequestID); err != nil {
		// Try parsing as a simple array
		var subscriptions []WebhookSubscription
		if jsonErr := json.Unmarshal(resp.Body, &subscriptions); jsonErr == nil {
			result.Data = subscriptions
		} else {
			return nil, err
		}
	}

	return &result, nil
}

// DeleteWebhookSubscription removes a webhook subscription by its ID.
// After deletion, your callback URL will no longer receive events for this subscription.
func (c *Client) DeleteWebhookSubscription(ctx context.Context, subscriptionID string) error {
	if subscriptionID == "" {
		return NewValidationError(400, "Subscription ID required", "subscriptionID cannot be empty", "subscription_id")
	}

	// Get the app ID from the config
	appID := c.config.ClientID

	c.mu.RLock()
	token := c.accessToken
	c.mu.RUnlock()

	// DELETE /{app-id}/subscriptions with the object parameter
	// The subscription ID is actually the object type for Meta's API (e.g., "instagram", "page")
	queryParams := url.Values{}
	queryParams.Set("object", subscriptionID)

	resp, err := c.httpClient.Do(&RequestOptions{
		Method:      "DELETE",
		Path:        fmt.Sprintf("/v1.0/%s/subscriptions", appID),
		QueryParams: queryParams,
	}, token)
	if err != nil {
		return err
	}

	// Parse the response
	var result struct {
		Success bool `json:"success"`
	}
	if err := safeJSONUnmarshal(resp.Body, &result, "delete webhook subscription", resp.RequestID); err != nil {
		return err
	}

	if !result.Success {
		return NewAPIError(resp.StatusCode, "Failed to delete webhook subscription", string(resp.Body), resp.RequestID)
	}

	return nil
}

// Compile-time check to ensure Client implements WebhookManager
var _ WebhookManager = (*Client)(nil)
