package threads

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// testLogger implements Logger interface for testing
type testLogger struct{}

func (l *testLogger) Debug(msg string, fields ...any) {}
func (l *testLogger) Info(msg string, fields ...any)  {}
func (l *testLogger) Warn(msg string, fields ...any)  {}
func (l *testLogger) Error(msg string, fields ...any) {}

// newTestClient creates a client configured to use the test server
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	config := &Config{
		ClientID:     "test-app-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "https://example.com/callback",
		Scopes:       []string{"threads_basic"},
		HTTPTimeout:  30 * time.Second,
		BaseURL:      server.URL,
		UserAgent:    "test-agent",
		Logger:       &testLogger{},
		RetryConfig: &RetryConfig{
			MaxRetries:    0, // No retries for tests
			InitialDelay:  1 * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
		},
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	// Set access token
	client.accessToken = "test-access-token"

	// Override the httpClient's baseURL to point to the test server
	// This is necessary because NewHTTPClient hardcodes the baseURL
	client.httpClient.baseURL = server.URL

	return client
}

func TestSubscribeWebhook_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		// Verify path contains app ID
		expectedPath := "/v1.0/test-app-id/subscriptions"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Verify content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/x-www-form-urlencoded" {
			t.Errorf("expected Content-Type application/x-www-form-urlencoded, got %s", contentType)
		}

		// Parse form data
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		// Verify form fields
		if r.Form.Get("object") != "user" {
			t.Errorf("expected object=user, got %s", r.Form.Get("object"))
		}
		if r.Form.Get("callback_url") != "https://example.com/webhook" {
			t.Errorf("expected callback_url=https://example.com/webhook, got %s", r.Form.Get("callback_url"))
		}
		if r.Form.Get("fields") != "mentions,publishes" {
			t.Errorf("expected fields=mentions,publishes, got %s", r.Form.Get("fields"))
		}
		if r.Form.Get("verify_token") != "my-verify-token" {
			t.Errorf("expected verify_token=my-verify-token, got %s", r.Form.Get("verify_token"))
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}

	client := newTestClient(t, handler)

	opts := &WebhookSubscribeOptions{
		CallbackURL: "https://example.com/webhook",
		VerifyToken: "my-verify-token",
		Fields:      []WebhookEventType{WebhookEventMentions, WebhookEventPublishes},
	}

	subscription, err := client.SubscribeWebhook(context.Background(), opts)
	if err != nil {
		t.Fatalf("SubscribeWebhook failed: %v", err)
	}

	if subscription == nil {
		t.Fatal("expected subscription, got nil")
	}

	if subscription.Object != "user" {
		t.Errorf("expected Object=user, got %s", subscription.Object)
	}

	if subscription.CallbackURL != "https://example.com/webhook" {
		t.Errorf("expected CallbackURL=https://example.com/webhook, got %s", subscription.CallbackURL)
	}

	if !subscription.Active {
		t.Error("expected Active=true")
	}

	if len(subscription.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(subscription.Fields))
	}
}

func TestSubscribeWebhook_NilOptions(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called with nil options")
	})

	_, err := client.SubscribeWebhook(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil options")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if validationErr.Field != "opts" {
		t.Errorf("expected field=opts, got %s", validationErr.Field)
	}
}

func TestSubscribeWebhook_EmptyCallbackURL(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called with empty callback URL")
	})

	opts := &WebhookSubscribeOptions{
		CallbackURL: "",
		Fields:      []WebhookEventType{WebhookEventMentions},
	}

	_, err := client.SubscribeWebhook(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error for empty callback URL")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if validationErr.Field != "callback_url" {
		t.Errorf("expected field=callback_url, got %s", validationErr.Field)
	}
}

func TestSubscribeWebhook_EmptyFields(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called with empty fields")
	})

	opts := &WebhookSubscribeOptions{
		CallbackURL: "https://example.com/webhook",
		Fields:      []WebhookEventType{},
	}

	_, err := client.SubscribeWebhook(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error for empty fields")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if validationErr.Field != "fields" {
		t.Errorf("expected field=fields, got %s", validationErr.Field)
	}
}

func TestSubscribeWebhook_APIError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Invalid callback URL",
				"type":    "OAuthException",
				"code":    100,
			},
		})
	}

	client := newTestClient(t, handler)

	opts := &WebhookSubscribeOptions{
		CallbackURL: "https://example.com/webhook",
		Fields:      []WebhookEventType{WebhookEventMentions},
	}

	_, err := client.SubscribeWebhook(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error from API")
	}
}

func TestSubscribeWebhook_SuccessFalse(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": false})
	}

	client := newTestClient(t, handler)

	opts := &WebhookSubscribeOptions{
		CallbackURL: "https://example.com/webhook",
		Fields:      []WebhookEventType{WebhookEventMentions},
	}

	_, err := client.SubscribeWebhook(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error when success=false")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if !strings.Contains(apiErr.Message, "Failed to create webhook subscription") {
		t.Errorf("expected message to contain 'Failed to create webhook subscription', got %s", apiErr.Message)
	}
}

func TestSubscribeWebhook_WithoutVerifyToken(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		// verify_token should not be set
		if r.Form.Get("verify_token") != "" {
			t.Errorf("expected no verify_token, got %s", r.Form.Get("verify_token"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}

	client := newTestClient(t, handler)

	opts := &WebhookSubscribeOptions{
		CallbackURL: "https://example.com/webhook",
		VerifyToken: "", // No verify token
		Fields:      []WebhookEventType{WebhookEventMentions},
	}

	_, err := client.SubscribeWebhook(context.Background(), opts)
	if err != nil {
		t.Fatalf("SubscribeWebhook failed: %v", err)
	}
}

func TestSubscribeWebhook_AllEventTypes(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		fields := r.Form.Get("fields")
		if fields != "mentions,publishes,deletes" {
			t.Errorf("expected fields=mentions,publishes,deletes, got %s", fields)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}

	client := newTestClient(t, handler)

	opts := &WebhookSubscribeOptions{
		CallbackURL: "https://example.com/webhook",
		Fields:      []WebhookEventType{WebhookEventMentions, WebhookEventPublishes, WebhookEventDeletes},
	}

	subscription, err := client.SubscribeWebhook(context.Background(), opts)
	if err != nil {
		t.Fatalf("SubscribeWebhook failed: %v", err)
	}

	if len(subscription.Fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(subscription.Fields))
	}
}

func TestListWebhookSubscriptions_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodGet {
			t.Errorf("expected GET method, got %s", r.Method)
		}

		// Verify path contains app ID
		expectedPath := "/v1.0/test-app-id/subscriptions"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Verify access_token in query params
		if r.URL.Query().Get("access_token") != "test-access-token" {
			t.Errorf("expected access_token=test-access-token, got %s", r.URL.Query().Get("access_token"))
		}

		// Return subscriptions
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := WebhookSubscriptionsResponse{
			Data: []WebhookSubscription{
				{
					ID:          "123456789",
					Object:      "user",
					CallbackURL: "https://example.com/webhook",
					Active:      true,
					Fields: []WebhookField{
						{Name: "mentions", Version: "v1.0"},
						{Name: "publishes", Version: "v1.0"},
					},
					CreatedTime: "2024-01-15T10:30:00+0000",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}

	client := newTestClient(t, handler)

	result, err := client.ListWebhookSubscriptions(context.Background())
	if err != nil {
		t.Fatalf("ListWebhookSubscriptions failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if len(result.Data) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(result.Data))
	}

	sub := result.Data[0]
	if sub.ID != "123456789" {
		t.Errorf("expected ID=123456789, got %s", sub.ID)
	}

	if sub.Object != "user" {
		t.Errorf("expected Object=user, got %s", sub.Object)
	}

	if sub.CallbackURL != "https://example.com/webhook" {
		t.Errorf("expected CallbackURL=https://example.com/webhook, got %s", sub.CallbackURL)
	}

	if !sub.Active {
		t.Error("expected Active=true")
	}

	if len(sub.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(sub.Fields))
	}
}

func TestListWebhookSubscriptions_Empty(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := WebhookSubscriptionsResponse{
			Data: []WebhookSubscription{},
		}
		_ = json.NewEncoder(w).Encode(response)
	}

	client := newTestClient(t, handler)

	result, err := client.ListWebhookSubscriptions(context.Background())
	if err != nil {
		t.Fatalf("ListWebhookSubscriptions failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if len(result.Data) != 0 {
		t.Errorf("expected 0 subscriptions, got %d", len(result.Data))
	}
}

func TestListWebhookSubscriptions_ArrayResponse(t *testing.T) {
	// Test fallback parsing when API returns an array directly instead of {data: []}
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return array directly (fallback format)
		subscriptions := []WebhookSubscription{
			{
				ID:          "987654321",
				Object:      "user",
				CallbackURL: "https://other.com/webhook",
				Active:      true,
			},
		}
		_ = json.NewEncoder(w).Encode(subscriptions)
	}

	client := newTestClient(t, handler)

	result, err := client.ListWebhookSubscriptions(context.Background())
	if err != nil {
		t.Fatalf("ListWebhookSubscriptions failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if len(result.Data) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(result.Data))
	}

	if result.Data[0].ID != "987654321" {
		t.Errorf("expected ID=987654321, got %s", result.Data[0].ID)
	}
}

func TestListWebhookSubscriptions_MultipleSubscriptions(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := WebhookSubscriptionsResponse{
			Data: []WebhookSubscription{
				{
					ID:          "111111",
					Object:      "user",
					CallbackURL: "https://example.com/webhook1",
					Active:      true,
					Fields:      []WebhookField{{Name: "mentions"}},
				},
				{
					ID:          "222222",
					Object:      "user",
					CallbackURL: "https://example.com/webhook2",
					Active:      false,
					Fields:      []WebhookField{{Name: "publishes"}},
				},
				{
					ID:          "333333",
					Object:      "user",
					CallbackURL: "https://example.com/webhook3",
					Active:      true,
					Fields:      []WebhookField{{Name: "deletes"}},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}

	client := newTestClient(t, handler)

	result, err := client.ListWebhookSubscriptions(context.Background())
	if err != nil {
		t.Fatalf("ListWebhookSubscriptions failed: %v", err)
	}

	if len(result.Data) != 3 {
		t.Fatalf("expected 3 subscriptions, got %d", len(result.Data))
	}

	// Verify each subscription
	expectedIDs := []string{"111111", "222222", "333333"}
	for i, sub := range result.Data {
		if sub.ID != expectedIDs[i] {
			t.Errorf("subscription %d: expected ID=%s, got %s", i, expectedIDs[i], sub.ID)
		}
	}

	// Verify active states
	if !result.Data[0].Active {
		t.Error("subscription 0 should be active")
	}
	if result.Data[1].Active {
		t.Error("subscription 1 should be inactive")
	}
	if !result.Data[2].Active {
		t.Error("subscription 2 should be active")
	}
}

func TestListWebhookSubscriptions_APIError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Invalid access token",
				"type":    "OAuthException",
				"code":    190,
			},
		})
	}

	client := newTestClient(t, handler)

	_, err := client.ListWebhookSubscriptions(context.Background())
	if err == nil {
		t.Fatal("expected error from API")
	}

	authErr, ok := err.(*AuthenticationError)
	if !ok {
		t.Fatalf("expected AuthenticationError, got %T", err)
	}

	if authErr.Code != 190 {
		t.Errorf("expected code=190, got %d", authErr.Code)
	}
}

func TestDeleteWebhookSubscription_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}

		// Verify path contains app ID
		expectedPath := "/v1.0/test-app-id/subscriptions"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}

	client := newTestClient(t, handler)

	err := client.DeleteWebhookSubscription(context.Background(), "user")
	if err != nil {
		t.Fatalf("DeleteWebhookSubscription failed: %v", err)
	}
}

func TestDeleteWebhookSubscription_EmptyID(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called with empty subscription ID")
	})

	err := client.DeleteWebhookSubscription(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty subscription ID")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if validationErr.Field != "subscription_id" {
		t.Errorf("expected field=subscription_id, got %s", validationErr.Field)
	}
}

func TestDeleteWebhookSubscription_NotFound(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Object with ID 'nonexistent' does not exist",
				"type":    "OAuthException",
				"code":    100,
			},
		})
	}

	client := newTestClient(t, handler)

	err := client.DeleteWebhookSubscription(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent subscription")
	}
}

func TestDeleteWebhookSubscription_SuccessFalse(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": false})
	}

	client := newTestClient(t, handler)

	err := client.DeleteWebhookSubscription(context.Background(), "user")
	if err == nil {
		t.Fatal("expected error when success=false")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if !strings.Contains(apiErr.Message, "Failed to delete webhook subscription") {
		t.Errorf("expected message to contain 'Failed to delete webhook subscription', got %s", apiErr.Message)
	}
}

func TestDeleteWebhookSubscription_Unauthorized(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "You do not have permission to delete this subscription",
				"type":    "OAuthException",
				"code":    200,
			},
		})
	}

	client := newTestClient(t, handler)

	err := client.DeleteWebhookSubscription(context.Background(), "user")
	if err == nil {
		t.Fatal("expected error for unauthorized deletion")
	}

	authErr, ok := err.(*AuthenticationError)
	if !ok {
		t.Fatalf("expected AuthenticationError, got %T", err)
	}

	if authErr.Code != 200 {
		t.Errorf("expected code=200, got %d", authErr.Code)
	}
}

func TestDeleteWebhookSubscription_ServerError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Fb-Request-Id", "req-12345")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Internal server error",
				"type":    "ServerException",
				"code":    500,
			},
		})
	}

	client := newTestClient(t, handler)

	err := client.DeleteWebhookSubscription(context.Background(), "user")
	if err == nil {
		t.Fatal("expected error for server error")
	}
}

func TestWebhookEventTypeConstants(t *testing.T) {
	// Verify the event type constants are correctly defined
	if WebhookEventMentions != "mentions" {
		t.Errorf("expected WebhookEventMentions=mentions, got %s", WebhookEventMentions)
	}

	if WebhookEventPublishes != "publishes" {
		t.Errorf("expected WebhookEventPublishes=publishes, got %s", WebhookEventPublishes)
	}

	if WebhookEventDeletes != "deletes" {
		t.Errorf("expected WebhookEventDeletes=deletes, got %s", WebhookEventDeletes)
	}
}

func TestWebhookSubscriptionStruct(t *testing.T) {
	sub := &WebhookSubscription{
		ID:          "test-id",
		Object:      "user",
		CallbackURL: "https://example.com/webhook",
		Fields: []WebhookField{
			{Name: "mentions", Version: "v1.0"},
		},
		Active:      true,
		CreatedTime: "2024-01-15T10:30:00+0000",
	}

	if sub.ID != "test-id" {
		t.Errorf("expected ID=test-id, got %s", sub.ID)
	}

	if sub.Object != "user" {
		t.Errorf("expected Object=user, got %s", sub.Object)
	}

	if sub.CallbackURL != "https://example.com/webhook" {
		t.Errorf("expected CallbackURL=https://example.com/webhook, got %s", sub.CallbackURL)
	}

	if !sub.Active {
		t.Error("expected Active=true")
	}

	if len(sub.Fields) != 1 {
		t.Errorf("expected 1 field, got %d", len(sub.Fields))
	}

	if sub.Fields[0].Name != "mentions" {
		t.Errorf("expected field name=mentions, got %s", sub.Fields[0].Name)
	}

	if sub.Fields[0].Version != "v1.0" {
		t.Errorf("expected field version=v1.0, got %s", sub.Fields[0].Version)
	}
}

func TestWebhookSubscribeOptions_Struct(t *testing.T) {
	opts := &WebhookSubscribeOptions{
		CallbackURL: "https://example.com/webhook",
		VerifyToken: "my-token",
		Fields:      []WebhookEventType{WebhookEventMentions, WebhookEventPublishes},
	}

	if opts.CallbackURL != "https://example.com/webhook" {
		t.Errorf("expected CallbackURL=https://example.com/webhook, got %s", opts.CallbackURL)
	}

	if opts.VerifyToken != "my-token" {
		t.Errorf("expected VerifyToken=my-token, got %s", opts.VerifyToken)
	}

	if len(opts.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(opts.Fields))
	}
}

func TestWebhookManagerInterface(t *testing.T) {
	// Verify Client implements WebhookManager interface
	var _ WebhookManager = (*Client)(nil)
}

func TestWebhookSubscriptionsResponse_WithPaging(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := WebhookSubscriptionsResponse{
			Data: []WebhookSubscription{
				{ID: "123", Object: "user", Active: true},
			},
			Paging: &Paging{
				Cursors: &PagingCursors{
					Before: "cursor-before",
					After:  "cursor-after",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}

	client := newTestClient(t, handler)

	result, err := client.ListWebhookSubscriptions(context.Background())
	if err != nil {
		t.Fatalf("ListWebhookSubscriptions failed: %v", err)
	}

	if result.Paging == nil {
		t.Fatal("expected paging info, got nil")
	}

	if result.Paging.Cursors.Before != "cursor-before" {
		t.Errorf("expected Before=cursor-before, got %s", result.Paging.Cursors.Before)
	}

	if result.Paging.Cursors.After != "cursor-after" {
		t.Errorf("expected After=cursor-after, got %s", result.Paging.Cursors.After)
	}
}
