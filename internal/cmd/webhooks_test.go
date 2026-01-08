package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/salmonumbrella/threads-go/internal/iocontext"
	"github.com/salmonumbrella/threads-go/internal/outfmt"
)

func TestWebhooksCmd_Structure(t *testing.T) {
	f := newTestFactory(t)
	cmd := NewWebhooksCmd(f)

	if cmd.Use != "webhooks" {
		t.Errorf("expected Use=webhooks, got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
}

func TestWebhooksCmd_Subcommands(t *testing.T) {
	f := newTestFactory(t)
	cmd := NewWebhooksCmd(f)

	expectedSubs := map[string]bool{
		"subscribe": true,
		"list":      true,
		"delete":    true,
	}

	for _, sub := range cmd.Commands() {
		name := sub.Name()
		if !expectedSubs[name] {
			t.Errorf("unexpected subcommand: %s", name)
		}
		delete(expectedSubs, name)
	}

	for name := range expectedSubs {
		t.Errorf("missing subcommand: %s", name)
	}
}

// Integration tests with mock HTTP server

func TestWebhooksList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle refresh_access_token calls
		if strings.Contains(r.URL.Path, "refresh") {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{"success": true}); err != nil {
				t.Errorf("failed to encode refresh response: %v", err)
			}
			return
		}

		// Return mock webhook subscriptions
		response := map[string]any{
			"data": []map[string]any{
				{
					"id":           "123456789",
					"object":       "user",
					"callback_url": "https://example.com/webhook",
					"fields": []map[string]string{
						{"name": "mentions"},
						{"name": "publishes"},
					},
					"active": true,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	f, io := newIntegrationTestFactory(t, server.URL)
	cmd := newWebhooksListCmd(f)

	ctx := context.Background()
	ctx = iocontext.WithIO(ctx, io)
	cmd.SetContext(ctx)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := io.Out.(*bytes.Buffer).String()
	if !strings.Contains(output, "example.com") {
		t.Errorf("expected callback URL in output: %s", output)
	}
	if !strings.Contains(output, "user") {
		t.Errorf("expected object type in output: %s", output)
	}
	if !strings.Contains(output, "mentions") {
		t.Errorf("expected 'mentions' field in output: %s", output)
	}
}

func TestWebhooksList_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle refresh_access_token calls
		if strings.Contains(r.URL.Path, "refresh") {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{"success": true}); err != nil {
				t.Errorf("failed to encode refresh response: %v", err)
			}
			return
		}

		// Return empty list
		response := map[string]any{
			"data": []map[string]any{},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	f, io := newIntegrationTestFactory(t, server.URL)
	cmd := newWebhooksListCmd(f)

	ctx := context.Background()
	ctx = iocontext.WithIO(ctx, io)
	cmd.SetContext(ctx)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := io.Out.(*bytes.Buffer).String()
	if !strings.Contains(output, "No webhook subscriptions found") {
		t.Errorf("expected empty message in output: %s", output)
	}
}

func TestWebhooksList_JSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle refresh_access_token calls
		if strings.Contains(r.URL.Path, "refresh") {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{"success": true}); err != nil {
				t.Errorf("failed to encode refresh response: %v", err)
			}
			return
		}

		// Return mock webhook subscriptions
		response := map[string]any{
			"data": []map[string]any{
				{
					"id":           "123456789",
					"object":       "user",
					"callback_url": "https://example.com/webhook",
					"fields": []map[string]string{
						{"name": "mentions"},
					},
					"active": true,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	f, io := newIntegrationTestFactory(t, server.URL)
	cmd := newWebhooksListCmd(f)

	ctx := context.Background()
	ctx = iocontext.WithIO(ctx, io)
	ctx = outfmt.WithFormat(ctx, "json")
	cmd.SetContext(ctx)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	// Verify output is valid JSON
	output := io.Out.(*bytes.Buffer).String()
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("output is not valid JSON: %v\nOutput: %s", err, output)
	}

	// Verify JSON contains expected fields
	data, ok := result["data"].([]any)
	if !ok || len(data) == 0 {
		t.Errorf("JSON output missing or empty data, got: %v", result)
	}
}

func TestWebhooksSubscribe_HTTPValidation(t *testing.T) {
	// Test that http:// URLs fail validation before making API call
	f := newTestFactory(t)
	cmd := newWebhooksSubscribeCmd(f)
	cmd.SetArgs([]string{"--url", "http://example.com/webhook", "--event", "mentions"})

	ctx := context.Background()
	ctx = iocontext.WithIO(ctx, f.IO)
	cmd.SetContext(ctx)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for http:// URL")
	}

	// Verify the error is about HTTPS
	errMsg := err.Error()
	if !strings.Contains(errMsg, "HTTPS") && !strings.Contains(errMsg, "https") {
		t.Errorf("expected HTTPS error, got: %v", err)
	}
}

func TestWebhooksSubscribe_MissingURL(t *testing.T) {
	f := newTestFactory(t)
	cmd := newWebhooksSubscribeCmd(f)
	cmd.SetArgs([]string{"--event", "mentions"})

	ctx := context.Background()
	ctx = iocontext.WithIO(ctx, f.IO)
	cmd.SetContext(ctx)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing URL")
	}

	// Verify it's a required flag error
	errMsg := err.Error()
	if !strings.Contains(errMsg, "required") && !strings.Contains(errMsg, "url") {
		t.Errorf("expected required flag error for URL, got: %v", err)
	}
}

func TestWebhooksSubscribe_MissingEvent(t *testing.T) {
	f := newTestFactory(t)
	cmd := newWebhooksSubscribeCmd(f)
	cmd.SetArgs([]string{"--url", "https://example.com/webhook"})

	ctx := context.Background()
	ctx = iocontext.WithIO(ctx, f.IO)
	cmd.SetContext(ctx)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing event")
	}

	// Verify it's a required flag error
	errMsg := err.Error()
	if !strings.Contains(errMsg, "required") && !strings.Contains(errMsg, "event") {
		t.Errorf("expected required flag error for event, got: %v", err)
	}
}

func TestWebhooksSubscribe_InvalidEvent(t *testing.T) {
	f := newTestFactory(t)
	cmd := newWebhooksSubscribeCmd(f)
	cmd.SetArgs([]string{"--url", "https://example.com/webhook", "--event", "invalid_event"})

	ctx := context.Background()
	ctx = iocontext.WithIO(ctx, f.IO)
	cmd.SetContext(ctx)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid event type")
	}

	// Verify the error is about invalid event
	errMsg := err.Error()
	if !strings.Contains(errMsg, "invalid_event") && !strings.Contains(errMsg, "Invalid") {
		t.Errorf("expected invalid event error, got: %v", err)
	}
}

func TestWebhooksSubscribe_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle refresh_access_token calls
		if strings.Contains(r.URL.Path, "refresh") {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{"success": true}); err != nil {
				t.Errorf("failed to encode refresh response: %v", err)
			}
			return
		}

		// The webhook subscribe endpoint returns {"success": true} on success
		response := map[string]any{
			"success": true,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	f, io := newIntegrationTestFactory(t, server.URL)
	cmd := newWebhooksSubscribeCmd(f)
	cmd.SetArgs([]string{"--url", "https://example.com/webhook", "--event", "mentions"})

	ctx := context.Background()
	ctx = iocontext.WithIO(ctx, io)
	cmd.SetContext(ctx)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := io.Out.(*bytes.Buffer).String()
	// Verify that success message and callback URL are displayed
	if !strings.Contains(output, "example.com") {
		t.Errorf("expected callback URL in output: %s", output)
	}
	if !strings.Contains(output, "successfully") || !strings.Contains(output, "created") {
		t.Errorf("expected success message in output: %s", output)
	}
}

func TestWebhooksList_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
		outputContains []string
	}{
		{
			name: "single subscription",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "refresh") {
					_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
					return
				}
				response := map[string]any{
					"data": []map[string]any{
						{
							"id":           "111",
							"object":       "user",
							"callback_url": "https://app1.example.com/webhook",
							"fields":       []map[string]string{{"name": "mentions"}},
							"active":       true,
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
			},
			wantErr:        false,
			outputContains: []string{"user", "app1.example.com", "mentions", "yes"},
		},
		{
			name: "multiple subscriptions",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "refresh") {
					_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
					return
				}
				response := map[string]any{
					"data": []map[string]any{
						{
							"id":           "111",
							"object":       "user",
							"callback_url": "https://app1.example.com/webhook",
							"fields":       []map[string]string{{"name": "mentions"}},
							"active":       true,
						},
						{
							"id":           "222",
							"object":       "user",
							"callback_url": "https://app2.example.com/webhook",
							"fields":       []map[string]string{{"name": "publishes"}},
							"active":       false,
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
			},
			wantErr:        false,
			outputContains: []string{"app1.example.com", "app2.example.com", "mentions", "publishes"},
		},
		{
			name: "API error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "refresh") {
					_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				errResp := map[string]any{
					"error": map[string]any{
						"message": "Invalid access token",
						"code":    190,
					},
				}
				_ = json.NewEncoder(w).Encode(errResp)
			},
			wantErr:     true,
			errContains: "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			f, io := newIntegrationTestFactory(t, server.URL)
			cmd := newWebhooksListCmd(f)

			ctx := context.Background()
			ctx = iocontext.WithIO(ctx, io)
			cmd.SetContext(ctx)

			err := cmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !strings.Contains(strings.ToLower(err.Error()), tt.errContains) {
					t.Errorf("expected error containing %q, got: %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := io.Out.(*bytes.Buffer).String()
			for _, contains := range tt.outputContains {
				if !strings.Contains(output, contains) {
					t.Errorf("output missing %q, got: %s", contains, output)
				}
			}
		})
	}
}
