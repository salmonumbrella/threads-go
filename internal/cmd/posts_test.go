package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

func TestPostsCmd_Structure(t *testing.T) {
	f := newTestFactory(t)
	cmd := NewPostsCmd(f)

	if cmd.Use != "posts" {
		t.Errorf("expected Use=posts, got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
}

func TestPostsCmd_Subcommands(t *testing.T) {
	f := newTestFactory(t)
	cmd := NewPostsCmd(f)

	expectedSubs := map[string]bool{
		"create":     true,
		"get":        true,
		"list":       true,
		"delete":     true,
		"carousel":   true,
		"quote":      true,
		"repost":     true,
		"unrepost":   true,
		"ghost-list": true,
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

func TestPostsCreateCmd_Flags(t *testing.T) {
	f := newTestFactory(t)
	cmd := newPostsCreateCmd(f)

	flags := []struct {
		name      string
		shorthand string
	}{
		{"text", "t"},
		{"text-file", ""},
		{"emit", ""},
		{"image", ""},
		{"video", ""},
		{"alt-text", ""},
		{"reply-to", ""},
		{"poll", ""},
		{"ghost", ""},
		{"topic", ""},
		{"location", ""},
		{"reply-control", ""},
		{"gif", ""},
	}

	for _, f := range flags {
		flag := cmd.Flag(f.name)
		if flag == nil {
			t.Errorf("missing flag: %s", f.name)
			continue
		}
		if f.shorthand != "" && flag.Shorthand != f.shorthand {
			t.Errorf("flag %s expected shorthand %q, got %q", f.name, f.shorthand, flag.Shorthand)
		}
	}
}

func TestPostsGetCmd_Structure(t *testing.T) {
	f := newTestFactory(t)
	cmd := newPostsGetCmd(f)

	if cmd.Use != "get [post-id]" {
		t.Errorf("expected Use='get [post-id]', got %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator for exactly 1 arg")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestPostsListCmd_Structure(t *testing.T) {
	f := newTestFactory(t)
	cmd := newPostsListCmd(f)

	if cmd.Use != "list" {
		t.Errorf("expected Use=list, got %s", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestPostsDeleteCmd_Structure(t *testing.T) {
	f := newTestFactory(t)
	cmd := newPostsDeleteCmd(f)

	if cmd.Use != "delete [post-id]" {
		t.Errorf("expected Use='delete [post-id]', got %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator for exactly 1 arg")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestPostsQuoteCmd_Structure(t *testing.T) {
	f := newTestFactory(t)
	cmd := newPostsQuoteCmd(f)

	if cmd.Use != "quote [post-id]" {
		t.Errorf("expected Use='quote [post-id]', got %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestPostsQuoteCmd_Flags(t *testing.T) {
	f := newTestFactory(t)
	cmd := newPostsQuoteCmd(f)

	flags := []string{"text", "image", "video"}
	for _, flag := range flags {
		if cmd.Flag(flag) == nil {
			t.Errorf("missing flag: %s", flag)
		}
	}
}

func TestPostsQuoteCmd_HasExample(t *testing.T) {
	f := newTestFactory(t)
	cmd := newPostsQuoteCmd(f)

	if cmd.Example == "" {
		t.Error("expected Example to be set for quote command")
	}
}

func TestPostsRepostCmd_Structure(t *testing.T) {
	f := newTestFactory(t)
	cmd := newPostsRepostCmd(f)

	if cmd.Use != "repost [post-id]" {
		t.Errorf("expected Use='repost [post-id]', got %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestPostsRepostCmd_HasExample(t *testing.T) {
	f := newTestFactory(t)
	cmd := newPostsRepostCmd(f)

	if cmd.Example == "" {
		t.Error("expected Example to be set for repost command")
	}
}

func TestPostsUnrepostCmd_Structure(t *testing.T) {
	f := newTestFactory(t)
	cmd := newPostsUnrepostCmd(f)

	if cmd.Use != "unrepost [repost-id]" {
		t.Errorf("expected Use='unrepost [repost-id]', got %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestPostsUnrepostCmd_HasExample(t *testing.T) {
	f := newTestFactory(t)
	cmd := newPostsUnrepostCmd(f)

	if cmd.Example == "" {
		t.Error("expected Example to be set for unrepost command")
	}
}

func TestPostsUnrepostCmd_HasLongDescription(t *testing.T) {
	f := newTestFactory(t)
	cmd := newPostsUnrepostCmd(f)

	if cmd.Long == "" {
		t.Error("expected Long description to be set for unrepost command")
	}
}

func TestPostsCarouselCmd_Structure(t *testing.T) {
	f := newTestFactory(t)
	cmd := newPostsCarouselCmd(f)

	if cmd.Use != "carousel" {
		t.Errorf("expected Use=carousel, got %s", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestPostsCarouselCmd_Flags(t *testing.T) {
	f := newTestFactory(t)
	cmd := newPostsCarouselCmd(f)

	flags := []string{"items", "text", "alt-text", "reply-to", "timeout"}
	for _, flag := range flags {
		if cmd.Flag(flag) == nil {
			t.Errorf("missing flag: %s", flag)
		}
	}

	itemsFlag := cmd.Flag("items")
	if itemsFlag == nil {
		t.Fatal("--items flag not found")
	}
}

func TestPostsCarouselCmd_TimeoutDefault(t *testing.T) {
	f := newTestFactory(t)
	cmd := newPostsCarouselCmd(f)

	timeoutFlag := cmd.Flag("timeout")
	if timeoutFlag == nil {
		t.Fatal("missing timeout flag")
	}

	if timeoutFlag.DefValue != "300" {
		t.Errorf("expected timeout default=300, got %s", timeoutFlag.DefValue)
	}
}

func TestPostsCarouselCmd_HasExample(t *testing.T) {
	f := newTestFactory(t)
	cmd := newPostsCarouselCmd(f)

	if cmd.Example == "" {
		t.Error("expected Example to be set for carousel command")
	}
}

func TestDetectMediaType_Image(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/image.jpg", "IMAGE"},
		{"https://example.com/image.jpeg", "IMAGE"},
		{"https://example.com/image.png", "IMAGE"},
		{"https://example.com/image.gif", "IMAGE"},
		{"https://example.com/image.webp", "IMAGE"},
		{"https://example.com/image.JPG", "IMAGE"},
		{"https://example.com/image.PNG", "IMAGE"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := detectMediaType(tt.url)
			if result != tt.expected {
				t.Errorf("detectMediaType(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestDetectMediaType_Video(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/video.mp4", "VIDEO"},
		{"https://example.com/video.mov", "VIDEO"},
		{"https://example.com/video.m4v", "VIDEO"},
		{"https://example.com/video.webm", "VIDEO"},
		{"https://example.com/video.MP4", "VIDEO"},
		{"https://example.com/video.MOV", "VIDEO"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := detectMediaType(tt.url)
			if result != tt.expected {
				t.Errorf("detectMediaType(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestDetectMediaType_WithQueryParams(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/image.jpg?width=100", "IMAGE"},
		{"https://example.com/video.mp4?quality=hd", "VIDEO"},
		{"https://example.com/file.png?token=abc123", "IMAGE"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := detectMediaType(tt.url)
			if result != tt.expected {
				t.Errorf("detectMediaType(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestDetectMediaType_DefaultToImage(t *testing.T) {
	tests := []string{
		"https://example.com/file",
		"https://example.com/file.txt",
		"https://example.com/file.pdf",
		"https://example.com/file.unknown",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			result := detectMediaType(url)
			if result != "IMAGE" {
				t.Errorf("detectMediaType(%q) = %q, want IMAGE (default)", url, result)
			}
		})
	}
}

func TestPostsGhostListCmd_Structure(t *testing.T) {
	f := newTestFactory(t)
	cmd := newPostsGhostListCmd(f)

	if cmd.Use != "ghost-list" {
		t.Errorf("expected Use='ghost-list', got %s", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}

	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

// Integration tests with mock HTTP server

func TestPostsGet_Success(t *testing.T) {
	// Create mock server that returns a post
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return success for any request (including refresh_access_token)
		post := map[string]any{
			"id":         "12345",
			"username":   "testuser",
			"media_type": "TEXT",
			"text":       "Hello, world!",
			"permalink":  "https://api.net/t/12345",
			"timestamp":  time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(post); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	f, io := newIntegrationTestFactory(t, server.URL)

	// Create and execute command
	cmd := newPostsGetCmd(f)
	cmd.SetArgs([]string{"12345"})

	ctx := context.Background()
	ctx = iocontext.WithIO(ctx, io)
	cmd.SetContext(ctx)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	// Verify output contains expected data
	output := io.Out.(*bytes.Buffer).String()
	if !strings.Contains(output, "12345") {
		t.Errorf("output missing post ID, got: %s", output)
	}
	if !strings.Contains(output, "testuser") {
		t.Errorf("output missing username, got: %s", output)
	}
	if !strings.Contains(output, "TEXT") {
		t.Errorf("output missing media type, got: %s", output)
	}
}

func TestPostsGet_NotFound(t *testing.T) {
	// Create mock server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		errResp := map[string]any{
			"error": map[string]any{
				"message": "Post not found",
				"code":    100,
				"type":    "OAuthException",
			},
		}
		if err := json.NewEncoder(w).Encode(errResp); err != nil {
			t.Errorf("failed to encode error response: %v", err)
		}
	}))
	defer server.Close()

	f, io := newIntegrationTestFactory(t, server.URL)

	// Create and execute command
	cmd := newPostsGetCmd(f)
	cmd.SetArgs([]string{"nonexistent"})

	ctx := context.Background()
	ctx = iocontext.WithIO(ctx, io)
	cmd.SetContext(ctx)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent post, got nil")
	}

	// Check that the error message indicates post not found
	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "failed") {
		t.Errorf("expected error to mention 'not found' or 'failed', got: %v", err)
	}
}

func TestPostsGet_JSONOutput(t *testing.T) {
	// Create mock server that returns a post
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		post := map[string]any{
			"id":         "12345",
			"username":   "testuser",
			"media_type": "TEXT",
			"text":       "Hello, world!",
			"permalink":  "https://api.net/t/12345",
			"timestamp":  time.Now().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(post); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	f, io := newIntegrationTestFactory(t, server.URL)

	// Create and execute command
	cmd := newPostsGetCmd(f)
	cmd.SetArgs([]string{"12345"})

	// Set JSON output format via context
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
	if result["id"] != "12345" {
		t.Errorf("JSON output missing or wrong id, got: %v", result["id"])
	}
	if result["username"] != "testuser" {
		t.Errorf("JSON output missing or wrong username, got: %v", result["username"])
	}
}

func TestPostsGet_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		postID         string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
		outputContains []string
	}{
		{
			name:   "successful retrieval",
			postID: "123456789",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				post := map[string]any{
					"id":         "123456789",
					"username":   "creator",
					"media_type": "TEXT",
					"text":       "Test post content",
					"permalink":  "https://api.net/t/123456789",
					"timestamp":  time.Now().Format(time.RFC3339),
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(post)
			},
			wantErr:        false,
			outputContains: []string{"123456789", "creator", "TEXT"},
		},
		{
			name:   "image post",
			postID: "img123",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				post := map[string]any{
					"id":         "img123",
					"username":   "photographer",
					"media_type": "IMAGE",
					"text":       "Check out this photo",
					"media_url":  "https://example.com/image.jpg",
					"permalink":  "https://api.net/t/img123",
					"timestamp":  time.Now().Format(time.RFC3339),
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(post)
			},
			wantErr:        false,
			outputContains: []string{"img123", "IMAGE", "photographer"},
		},
		{
			name:   "video post",
			postID: "vid456",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				post := map[string]any{
					"id":         "vid456",
					"username":   "videographer",
					"media_type": "VIDEO",
					"text":       "New video!",
					"media_url":  "https://example.com/video.mp4",
					"permalink":  "https://api.net/t/vid456",
					"timestamp":  time.Now().Format(time.RFC3339),
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(post)
			},
			wantErr:        false,
			outputContains: []string{"vid456", "VIDEO", "videographer"},
		},
		{
			name:   "post not found",
			postID: "notfound",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				// For refresh_access_token, return success; for post request, return 404
				if strings.Contains(r.URL.Path, "refresh") {
					_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
					return
				}
				w.WriteHeader(http.StatusNotFound)
				errResp := map[string]any{
					"error": map[string]any{
						"message": "Post not found",
						"code":    100,
					},
				}
				_ = json.NewEncoder(w).Encode(errResp)
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:   "unauthorized access",
			postID: "private123",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				// For refresh_access_token, return success; for post request, return 403
				if strings.Contains(r.URL.Path, "refresh") {
					_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
					return
				}
				w.WriteHeader(http.StatusForbidden)
				errResp := map[string]any{
					"error": map[string]any{
						"message": "Access denied",
						"code":    200,
					},
				}
				_ = json.NewEncoder(w).Encode(errResp)
			},
			wantErr:     true,
			errContains: "denied",
		},
		{
			name:   "server error",
			postID: "server500",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				// For refresh_access_token, return success; for post request, return 500
				if strings.Contains(r.URL.Path, "refresh") {
					_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				errResp := map[string]any{
					"error": map[string]any{
						"message": "Internal server error",
						"code":    500,
					},
				}
				_ = json.NewEncoder(w).Encode(errResp)
			},
			wantErr:     true,
			errContains: "api", // Matches "API is experiencing issues"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			f, io := newIntegrationTestFactory(t, server.URL)

			cmd := newPostsGetCmd(f)
			cmd.SetArgs([]string{tt.postID})

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

func TestPostsCreate_EmitID_Text(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/refresh_access_token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "refreshed-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
			return
		case "/12345/threads":
			// create container
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "c1"})
			return
		case "/c1":
			// container status
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":            "c1",
				"status":        "FINISHED",
				"error_message": "",
			})
			return
		case "/12345/threads_publish":
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "p1"})
			return
		case "/p1":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":                 "p1",
				"permalink":          "https://www.threads.net/t/p1",
				"timestamp":          time.Now().UTC().Format(time.RFC3339),
				"username":           "testuser",
				"media_product_type": "THREADS",
				"is_reply":           false,
			})
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	f, io := newIntegrationTestFactory(t, server.URL)
	ctx := context.Background()
	ctx = iocontext.WithIO(ctx, io)
	ctx = outfmt.WithFormat(ctx, "text")

	cmd := newPostsCreateCmd(f)
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--text", "hi", "--emit", "id"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("posts create --emit id failed: %v", err)
	}

	out := io.Out.(*bytes.Buffer).String()
	if out != "p1\n" {
		t.Fatalf("expected id p1, got %q", out)
	}
}

func TestPostsRepost_EmitID_Text(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/refresh_access_token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "refreshed-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
			return
		case "/orig/repost":
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "rp1"})
			return
		case "/rp1":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":                 "rp1",
				"permalink":          "https://www.threads.net/t/rp1",
				"timestamp":          time.Now().UTC().Format(time.RFC3339),
				"username":           "testuser",
				"media_product_type": "THREADS",
				"is_reply":           false,
			})
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	f, io := newIntegrationTestFactory(t, server.URL)
	ctx := context.Background()
	ctx = iocontext.WithIO(ctx, io)
	ctx = outfmt.WithFormat(ctx, "text")

	cmd := newPostsRepostCmd(f)
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"orig", "--emit", "id"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("posts repost --emit id failed: %v", err)
	}

	out := io.Out.(*bytes.Buffer).String()
	if out != "rp1\n" {
		t.Fatalf("expected repost id rp1, got %q", out)
	}
}

func TestPostsGet_AcceptsURLAndPrefixedIDs(t *testing.T) {
	var gotPaths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/refresh_access_token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "refreshed-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
			return
		}

		gotPaths = append(gotPaths, r.URL.Path)
		post := map[string]any{
			"id":         strings.TrimPrefix(r.URL.Path, "/"),
			"username":   "testuser",
			"media_type": "TEXT",
			"text":       "Hello",
			"permalink":  "https://www.threads.net/t/" + strings.TrimPrefix(r.URL.Path, "/"),
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(post)
	}))
	defer server.Close()

	f, io := newIntegrationTestFactory(t, server.URL)

	ctx := context.Background()
	ctx = iocontext.WithIO(ctx, io)
	cmd := newPostsGetCmd(f)
	cmd.SetContext(ctx)

	cases := []string{
		"12345",
		"#12345",
		"post:12345",
		"https://www.threads.net/t/12345",
		"https://api.net/t/12345",
	}
	for _, c := range cases {
		io.Out.(*bytes.Buffer).Reset()
		io.ErrOut.(*bytes.Buffer).Reset()

		cmd2 := newPostsGetCmd(f)
		cmd2.SetContext(ctx)
		cmd2.SetArgs([]string{c})
		if err := cmd2.Execute(); err != nil {
			t.Fatalf("posts get %q failed: %v", c, err)
		}
	}

	// All requests should be for the extracted ID.
	for _, p := range gotPaths {
		if p == "/refresh_access_token" {
			continue
		}
		if p != "/12345" {
			t.Fatalf("expected request path /12345, got %q", p)
		}
	}
}
